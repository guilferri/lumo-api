package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mxschmitt/playwright-go"
)

type Driver struct {
	pw      playwright.Playwright
	browser playwright.Browser
	page    playwright.Page
	busy    bool
}

// NewDriver creates (and starts) a Playwright instance.
// It expects an `auth.json` file in the working directory that contains
// a valid Chromium storage state for a logged‑in Lumo session.
func NewDriver() (*Driver, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("playwright launch failed: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--disable-gpu",
			"--no-sandbox",
			"--disable-dev-shm-usage",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("chromium launch failed: %w", err)
	}

	ctx := context.Background()
	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("new page: %w", err)
	}

	// -----------------------------------------------------------------
	// Load authentication state – if missing we fall back to manual login.
	// -----------------------------------------------------------------
	if _, statErr := os.Stat("auth.json"); statErr == nil {
		// Load stored cookies / local storage
		if err = page.Context().AddCookiesFromFile("auth.json"); err != nil {
			return nil, fmt.Errorf("load auth.json: %w", err)
		}
	} else {
		// First‑time run – open login page and wait for user to finish.
		fmt.Println("⚡ No auth.json found – opening Lumo login page.")
		if _, err = page.Goto("https://lumo.proton.me/login", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
		}); err != nil {
			return nil, fmt.Errorf("goto login: %w", err)
		}
		fmt.Print("🔐 After you log in, press ENTER to continue...")
		fmt.Scanln()

		// Persist the storage state for next runs.
		if err = page.Context().StorageState(&playwright.BrowserContextStorageStateOptions{
			Path: playwright.String("auth.json"),
		}); err != nil {
			return nil, fmt.Errorf("save auth.json: %w", err)
		}
	}

	// Navigate to the main chat UI.
	if _, err = page.Goto("https://lumo.proton.me/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return nil, fmt.Errorf("goto main UI: %w", err)
	}

	// Wait for the chat input to become available.
	if _, err = page.WaitForSelector("textarea[data-testid='chat-input']", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(30000),
	}); err != nil {
		return nil, fmt.Errorf("chat input not found: %w", err)
	}

	log.Println("✅ Lumo browser session ready")
	return &Driver{
		pw:      pw,
		browser: browser,
		page:    page,
	}, nil
}

// Close shuts down Playwright cleanly.
func (d *Driver) Close() error {
	if d.page != nil {
		_ = d.page.Close()
	}
	if d.browser != nil {
		_ = d.browser.Close()
	}
	if d.pw != nil {
		_ = d.pw.Stop()
	}
	return nil
}

// Prompt sends a prompt to Lumo and returns the assistant’s answer.
// It respects the `webSearch` flag by toggling the UI button.
func (d *Driver) Prompt(ctx context.Context, prompt string, webSearch bool) (string, error) {
	if d.busy {
		return "", fmt.Errorf("driver busy")
	}
	d.busy = true
	defer func() { d.busy = false }()

	// -----------------------------------------------------------------
	// Web‑search toggle – the button has a deterministic test‑id.
	// -----------------------------------------------------------------
	if webSearch {
		if err := d.toggleWebSearch(true); err != nil {
			return "", err
		}
	} else {
		if err := d.toggleWebSearch(false); err != nil {
			return "", err
		}
	}

	// Focus the textarea and type the prompt.
	textarea := d.page.Locator("textarea[data-testid='chat-input']")
	if err := textarea.Fill(""); err != nil {
		return "", fmt.Errorf("clear textarea: %w", err)
	}
	if err := textarea.Type(prompt, playwright.ElementHandleTypeOptions{
		Delay: playwright.Float(0), // 0 ms for speed; increase if you hit anti‑bot detection.
	}); err != nil {
		return "", fmt.Errorf("type prompt: %w", err)
	}
	if err := textarea.Press("Enter"); err != nil {
		return "", fmt.Errorf("press enter: %w", err)
	}

	// -----------------------------------------------------------------
	// Wait for the assistant’s message bubble to appear.
	// The last bubble with data-testid='assistant-message' holds the answer.
	// -----------------------------------------------------------------
	respLocator := d.page.Locator("div[data-testid='assistant-message']").Last()
	var answer string
	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timeout waiting for answer")
		default:
			text, err := respLocator.InnerText()
			if err == nil && text != "" && text != "…" {
				answer = text
				// Double‑check that the text has stabilized.
				time.Sleep(300 * time.Millisecond)
				again, _ := respLocator.InnerText()
				if again == answer {
					return answer, nil
				}
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// toggleWebSearch ensures the web‑search button matches the desired state.
func (d *Driver) toggleWebSearch(enable bool) error {
	btn, err := d.page.QuerySelector("button[data-testid='web-search-toggle']")
	if err != nil || btn == nil {
		// Button may not exist in older UI versions – ignore silently.
		return nil
	}
	class, _ := btn.GetAttribute("class")
	isActive := false
	if class != nil && strings.Contains(*class, "is-active") {
		isActive = true
	}
	if enable && !isActive {
		return btn.Click()
	}
	if !enable && isActive {
		return btn.Click()
	}
	return nil
}
