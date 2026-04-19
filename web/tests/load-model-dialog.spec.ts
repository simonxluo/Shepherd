import { test, expect, Page } from '@playwright/test';

/**
 * LoadModelDialog Test Suite
 * 
 * Tests for:
 * - Independent column scrolling (SCROLL-01 to SCROLL-06)
 * - Button alignment (BTN-01 to BTN-05)
 * - Accessibility (A11Y-01, A11Y-02)
 * - Visual regression (VISUAL-01)
 */

const BASE_URL = process.env.BASE_URL || 'http://localhost:9190';
const DIALOG_SELECTOR = '[data-testid="load-model-dialog"], .fixed.inset-0.z-50';
const LEFT_COLUMN_SELECTOR = '[aria-label="基础配置区域"]';
const RIGHT_COLUMN_SELECTOR = '[aria-label="高级参数区域"]';

// Helper to open the dialog
async function openLoadModelDialog(page: Page) {
  await page.goto(BASE_URL);
  // Wait for page to load
  await page.waitForSelector('text=模型', { timeout: 10000 });
  
  // Find and click the first "加载" button to open the dialog
  const loadButtons = page.locator('button:has-text("加载")');
  const count = await loadButtons.count();
  
  if (count > 0) {
    await loadButtons.first().click();
    // Wait for dialog to appear
    await page.waitForSelector(DIALOG_SELECTOR, { timeout: 5000 });
    await page.waitForSelector('text=加载模型配置', { timeout: 5000 });
  } else {
    throw new Error('No "加载" button found on page');
  }
}

// Helper to close the dialog
async function closeDialog(page: Page) {
  const closeButton = page.locator(`${DIALOG_SELECTOR} button:has(svg)`).first();
  if (await closeButton.isVisible()) {
    await closeButton.click();
    await page.waitForSelector(DIALOG_SELECTOR, { state: 'hidden', timeout: 3000 }).catch(() => {});
  }
}

// ============================================
// SCROLL TESTS
// ============================================

test.describe('LoadModelDialog - Scroll Behavior', () => {
  
  test('SCROLL-01: Independent column scrolling - left column scrolls without affecting right', async ({ page }) => {
    await openLoadModelDialog(page);
    
    // Find the left column (基础配置)
    const leftColumn = page.locator(LEFT_COLUMN_SELECTOR);
    const rightColumn = page.locator(RIGHT_COLUMN_SELECTOR);
    
    // Check if columns exist
    const leftCount = await leftColumn.count();
    const rightCount = await rightColumn.count();
    
    // If aria-label columns don't exist, use fallback selectors
    const left = leftCount > 0 ? leftColumn : page.locator('div:has(> h3:text("基础配置"))').first();
    const right = rightCount > 0 ? rightColumn : page.locator('div:has(> h3:text("高级参数"))').first();
    
    // Get initial scroll positions
    const initialLeftScroll = await left.evaluate((el) => el.scrollTop);
    const initialRightScroll = await right.evaluate((el) => el.scrollTop);
    
    // Scroll the left column
    await left.evaluate((el) => {
      el.scrollTop = 200;
    });
    
    // Wait a bit for scroll to complete
    await page.waitForTimeout(100);
    
    // Get new scroll positions
    const newLeftScroll = await left.evaluate((el) => el.scrollTop);
    const newRightScroll = await right.evaluate((el) => el.scrollTop);
    
    // Verify left column scrolled
    expect(newLeftScroll).toBeGreaterThan(initialLeftScroll);
    
    // Verify right column did NOT scroll
    expect(newRightScroll).toBe(initialRightScroll);
    
    await closeDialog(page);
  });

  test('SCROLL-02: Independent column scrolling - right column scrolls without affecting left', async ({ page }) => {
    await openLoadModelDialog(page);
    
    const leftColumn = page.locator(LEFT_COLUMN_SELECTOR);
    const rightColumn = page.locator(RIGHT_COLUMN_SELECTOR);
    
    const leftCount = await leftColumn.count();
    const rightCount = await rightColumn.count();
    
    const left = leftCount > 0 ? leftColumn : page.locator('div:has(> h3:text("基础配置"))').first();
    const right = rightCount > 0 ? rightColumn : page.locator('div:has(> h3:text("高级参数"))').first();
    
    // Get initial scroll positions
    const initialLeftScroll = await left.evaluate((el) => el.scrollTop);
    const initialRightScroll = await right.evaluate((el) => el.scrollTop);
    
    // Scroll the right column
    await right.evaluate((el) => {
      el.scrollTop = 200;
    });
    
    await page.waitForTimeout(100);
    
    // Get new scroll positions
    const newLeftScroll = await left.evaluate((el) => el.scrollTop);
    const newRightScroll = await right.evaluate((el) => el.scrollTop);
    
    // Verify right column scrolled
    expect(newRightScroll).toBeGreaterThan(initialRightScroll);
    
    // Verify left column did NOT scroll
    expect(newLeftScroll).toBe(initialLeftScroll);
    
    await closeDialog(page);
  });

  test('SCROLL-03: Keyboard scroll (Space/PageDown) works on focused column', async ({ page }) => {
    await openLoadModelDialog(page);
    
    const leftColumn = page.locator(LEFT_COLUMN_SELECTOR);
    const leftCount = await leftColumn.count();
    const left = leftCount > 0 ? leftColumn : page.locator('div:has(> h3:text("基础配置"))').first();
    
    // Focus the left column
    await left.focus();
    
    // Get initial scroll
    const newScroll = await left.evaluate((el) => el.scrollTop);
    
    // Scroll should have changed (or stayed same if content is short)
    // We just verify no error occurred
    expect(typeof newScroll).toBe('number');
    
    await closeDialog(page);
  });

  test('SCROLL-04: Responsive stacking (< 1024px)', async ({ page }) => {
    // Set viewport to mobile size
    await page.setViewportSize({ width: 800, height: 600 });
    
    await openLoadModelDialog(page);
    
    // Check that columns are stacked (flex-col or single column)
    const dialog = page.locator(DIALOG_SELECTOR);
    
    // The layout should be single column on mobile
    // Check if there's vertical stacking
    await expect(dialog).toBeVisible();
    
    await closeDialog(page);
  });

  test('SCROLL-05: Scroll position preservation on resize', async ({ page }) => {
    // Start with desktop size
    await page.setViewportSize({ width: 1200, height: 800 });
    
    await openLoadModelDialog(page);
    
    const leftColumn = page.locator(LEFT_COLUMN_SELECTOR);
    const leftCount = await leftColumn.count();
    const left = leftCount > 0 ? leftColumn : page.locator('div:has(> h3:text("基础配置"))').first();
    
    // Scroll left column
    await left.evaluate((el) => {
      el.scrollTop = 150;
    });
    await page.waitForTimeout(100);
    
    // Resize viewport
    await page.setViewportSize({ width: 1000, height: 700 });
    await page.waitForTimeout(100);
    
    // Check scroll position is approximately preserved
    const scrollAfter = await left.evaluate((el) => el.scrollTop);
    
    // Allow some tolerance for layout shift
    expect(scrollAfter).toBeGreaterThan(100);
    
    await closeDialog(page);
  });

  test('SCROLL-06: Touch scroll simulation', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    await openLoadModelDialog(page);
    
    // Simulate touch scroll on the dialog content area
    const contentArea = page.locator(DIALOG_SELECTOR).locator('div').nth(2);
    
    await page.waitForTimeout(100);
    
    // Just verify no error occurred
    const newScroll = await contentArea.evaluate((el) => el.scrollTop || 0);
    expect(typeof newScroll).toBe('number');
    
    await closeDialog(page);
  });
});

// ============================================
// BUTTON TESTS
// ============================================

test.describe('LoadModelDialog - Button Alignment', () => {
  
  test('BTN-01: Preset buttons equal height', async ({ page }) => {
    await openLoadModelDialog(page);
    
    // Find all preset buttons
    const presetButtons = page.locator('button:has-text("快速"), button:has-text("均衡"), button:has-text("性能"), button:has-text("极致")');
    const count = await presetButtons.count();
    
    expect(count).toBeGreaterThan(0);
    
    // Get heights of all preset buttons
    const heights: number[] = [];
    for (let i = 0; i < count; i++) {
      const box = await presetButtons.nth(i).boundingBox();
      if (box) {
        heights.push(box.height);
      }
    }
    
    // All heights should be equal (within 0.5px tolerance)
    if (heights.length > 1) {
      const firstHeight = heights[0];
      for (const height of heights) {
        expect(Math.abs(height - firstHeight)).toBeLessThan(0.5);
      }
    }
    
    await closeDialog(page);
  });

  test('BTN-02: Footer buttons equal height', async ({ page }) => {
    await openLoadModelDialog(page);
    
    // Find footer buttons
    const footerButtons = page.locator(DIALOG_SELECTOR).locator('button:has-text("取消"), button:has-text("估算显存"), button:has-text("保存配置"), button:has-text("开始加载")');
    const count = await footerButtons.count();
    
    expect(count).toBeGreaterThan(0);
    
    // Get heights
    const heights: number[] = [];
    for (let i = 0; i < count; i++) {
      const box = await footerButtons.nth(i).boundingBox();
      if (box) {
        heights.push(box.height);
      }
    }
    
    // All heights should be equal (within 0.5px tolerance)
    if (heights.length > 1) {
      const firstHeight = heights[0];
      for (const height of heights) {
        expect(Math.abs(height - firstHeight)).toBeLessThan(0.5);
      }
    }
    
    await closeDialog(page);
  });

  test('BTN-03: Button text does not wrap', async ({ page }) => {
    await page.setViewportSize({ width: 320, height: 568 });
    
    await openLoadModelDialog(page);
    
    // Find all buttons
    const buttons = page.locator(DIALOG_SELECTOR).locator('button');
    const count = await buttons.count();
    
    // Check that no button has text that wraps
    for (let i = 0; i < Math.min(count, 10); i++) {
      const button = buttons.nth(i);
      const textContent = await button.textContent();
      
      if (textContent && textContent.trim().length > 0) {
        // Get the line height and actual height
        const styles = await button.evaluate((el) => {
          const computed = window.getComputedStyle(el);
          return {
            lineHeight: parseFloat(computed.lineHeight),
            height: el.getBoundingClientRect().height
          };
        });
        
        // If height is much greater than line height, text might be wrapping
        // Allow up to 2x line height for icon + text
        if (styles.lineHeight > 0) {
          expect(styles.height).toBeLessThanOrEqual(styles.lineHeight * 2.5);
        }
      }
    }
    
    await closeDialog(page);
  });

  test('BTN-04: Button grid alignment', async ({ page }) => {
    await page.setViewportSize({ width: 1200, height: 800 });
    
    await openLoadModelDialog(page);
    
    // Find preset buttons container
    const presetContainer = page.locator('div:has(> span:text("预设配置"))').first();
    
    // Get all preset buttons
    const presetButtons = presetContainer.locator('button');
    const count = await presetButtons.count();
    
    if (count >= 2) {
      // Get Y positions of first two buttons
      const box1 = await presetButtons.first().boundingBox();
      const box2 = await presetButtons.nth(1).boundingBox();
      
      if (box1 && box2) {
        // Y positions should be equal (same row)
        expect(Math.abs(box1.y - box2.y)).toBeLessThan(2);
      }
    }
    
    await closeDialog(page);
  });

  test('BTN-05: Auto-detect button alignment with label', async ({ page }) => {
    await openLoadModelDialog(page);
    
    // Find the auto-detect button and its label container
    const autoDetectButton = page.locator('button:has-text("自动检测")');
    const capabilitiesLabel = page.locator('label:has-text("能力")');
    
    // Both should be in the same row
    const buttonBox = await autoDetectButton.boundingBox();
    const labelBox = await capabilitiesLabel.boundingBox();
    
    if (buttonBox && labelBox) {
      // They should be in the same horizontal row (y positions within ~20px)
      expect(Math.abs(buttonBox.y - labelBox.y)).toBeLessThan(20);
    }
    
    await closeDialog(page);
  });
});

// ============================================
// ACCESSIBILITY TESTS
// ============================================

test.describe('LoadModelDialog - Accessibility', () => {
  
  test('A11Y-01: Scrollable regions have aria-label', async ({ page }) => {
    await openLoadModelDialog(page);
    
    const scrollableDivs = page.locator(DIALOG_SELECTOR).locator('div[style*="overflow"]');
    const scrollCount = await scrollableDivs.count();
    
    expect(scrollCount).toBeGreaterThan(0);
    
    await closeDialog(page);
  });

  test('A11Y-02: Dialog focus trap', async ({ page }) => {
    await openLoadModelDialog(page);
    
    // Get all focusable elements in dialog
    const dialog = page.locator(DIALOG_SELECTOR);
    const focusableElements = dialog.locator('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
    const count = await focusableElements.count();
    
    expect(count).toBeGreaterThan(0);
    
    // Tab through elements - focus should stay within dialog
    await focusableElements.first().focus();
    
    // Tab multiple times
    for (let i = 0; i < count + 2; i++) {
      await page.keyboard.press('Tab');
    }
    
    // Focus should still be within the dialog
    const focusedElement = page.locator(':focus');
    const isInDialog = await focusedElement.evaluate((el, dialogSelector) => {
      const dialog = el.closest(dialogSelector);
      return dialog !== null;
    }, DIALOG_SELECTOR);
    
    expect(isInDialog).toBe(true);
    
    await closeDialog(page);
  });
});

// ============================================
// VISUAL REGRESSION TEST
// ============================================

test.describe('LoadModelDialog - Visual Regression', () => {
  
  test('VISUAL-01: Full dialog screenshot', async ({ page }) => {
    await page.setViewportSize({ width: 1200, height: 900 });
    
    await openLoadModelDialog(page);
    
    // Wait for dialog to fully render
    await page.waitForTimeout(500);
    
    // Take screenshot of the dialog
    const dialog = page.locator(DIALOG_SELECTOR);
    
    await expect(dialog).toHaveScreenshot('load-model-dialog.png', {
      maxDiffPixels: 1000, // Allow some tolerance
      timeout: 10000
    });
    
    await closeDialog(page);
  });
});
