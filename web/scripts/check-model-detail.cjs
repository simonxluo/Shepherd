const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({
    headless: true
  });

  const page = await browser.newPage();

  try {
    console.log('正在打开页面: http://localhost:3030');
    await page.goto('http://localhost:3030', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    // 截图首页
    await page.screenshot({ path: '/tmp/model-page.png', fullPage: true });
    console.log('✓ 首页截图已保存: /tmp/model-page.png');

    // 查找模型详情按钮
    const detailButtons = await page.$$('button[title="模型详情"], button[aria-label="模型详情"], button:has-text("详情")');
    console.log(`找到 ${detailButtons.length} 个可能的详情按钮`);

    // 也检查 Info 图标
    const infoIcons = await page.$$('button svg.lucide-info');
    console.log(`找到 ${infoIcons.length} 个 Info 图标`);

    if (detailButtons.length > 0 || infoIcons.length > 0) {
      console.log('点击第一个按钮/图标...');
      const target = detailButtons[0] || infoIcons[0];
      await target.click();
      await page.waitForTimeout(1000);

      // 截图对话框
      await page.screenshot({ path: '/tmp/model-detail-dialog.png', fullPage: true });
      console.log('✓ 详情对话框截图: /tmp/model-detail-dialog.png');

      // 检查对话框是否可见
      const dialogVisible = await page.isVisible('.fixed.inset-0.z-50').catch(() => false);
      console.log(`对话框可见: ${dialogVisible}`);

      if (dialogVisible) {
        // 获取对话框标题
        const title = await page.$eval('.fixed.inset-0.z-50 h2', el => el.textContent).catch(() => 'N/A');
        console.log(`对话框标题: ${title}`);
      }
    } else {
      console.log('⚠ 未找到详情按钮');
      const modelCards = await page.$$('.group.flex.items-center');
      console.log(`找到 ${modelCards.length} 个模型卡片`);
    }

    // 检查控制台错误
    const errors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    await page.waitForTimeout(2000);

    if (errors.length > 0) {
      console.log('\n控制台错误:');
      errors.forEach(e => console.log(`  - ${e}`));
    } else {
      console.log('\n✓ 没有控制台错误');
    }

  } catch (error) {
    console.error('检查出错:', error.message);
  } finally {
    await browser.close();
    console.log('\n检查完成!');
  }
})();
