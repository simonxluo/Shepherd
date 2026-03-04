import { chromium, type Browser, type Page } from 'playwright';

async function checkModelDetailPage() {
  const browser: Browser = await chromium.launch({
    headless: false,
    slowMo: 500 // 放慢操作以便观察
  });

  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 }
  });

  const page: Page = await context.newPage();

  try {
    console.log('正在打开页面...');
    await page.goto('http://localhost:3030');

    console.log('等待页面加载...');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // 截图首页
    await page.screenshot({ path: '/tmp/model-page.png', fullPage: true });
    console.log('✓ 首页截图已保存到 /tmp/model-page.png');

    // 查找模型详情按钮
    console.log('查找模型详情按钮...');
    const detailButtons = await page.locator('button[title="模型详情"]').all();
    console.log(`找到 ${detailButtons.length} 个模型详情按钮`);

    if (detailButtons.length > 0) {
      // 点击第一个详情按钮
      console.log('点击第一个模型详情按钮...');
      await detailButtons[0].click();

      // 等待对话框出现
      await page.waitForTimeout(1000);

      // 截图对话框
      await page.screenshot({ path: '/tmp/model-detail-dialog.png', fullPage: true });
      console.log('✓ 模型详情对话框截图已保存到 /tmp/model-detail-dialog.png');

      // 检查对话框内容
      const dialogVisible = await page.locator('.fixed.inset-0.z-50').isVisible();
      console.log(`对话框可见: ${dialogVisible}`);

      // 检查详情内容
      const detailSections = await page.locator('.bg-muted\\/30').all();
      console.log(`找到 ${detailSections.length} 个详情分组`);

      // 尝试关闭对话框
      console.log('关闭对话框...');
      const closeButton = page.locator('button:has-text("关闭")').first();
      if (await closeButton.isVisible()) {
        await closeButton.click();
        await page.waitForTimeout(500);
        console.log('✓ 对话框已关闭');
      }
    } else {
      console.log('⚠ 未找到模型详情按钮，检查 ModelCard 组件...');
      const modelCards = await page.locator('.group.flex.items-center').all();
      console.log(`找到 ${modelCards.length} 个模型卡片`);
    }

    // 检查控制台错误
    const logs: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        logs.push(msg.text());
      }
    });

    if (logs.length > 0) {
      console.log('\n控制台错误:');
      logs.forEach(log => console.log(`  - ${log}`));
    } else {
      console.log('\n✓ 没有控制台错误');
    }

    // 保持浏览器打开一段时间以便观察
    console.log('\n浏览器将保持打开 10 秒以便观察...');
    await page.waitForTimeout(10000);

  } catch (error) {
    console.error('检查过程中出错:', error);
  } finally {
    await browser.close();
    console.log('\n检查完成！');
  }
}

checkModelDetailPage().catch(console.error);
