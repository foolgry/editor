#!/usr/bin/env node

/**
 * wxmd-cli - 公众号 Markdown 编辑器 CLI
 * Agent-First 命令行工具
 */

const { program } = require('commander');
const pkg = require('../package.json');

// 导入命令
const { registerTypesetCommand } = require('./commands/typeset');
const { registerShareCommand } = require('./commands/share');
const { registerStylesCommand } = require('./commands/styles');
const { registerDoctorCommand } = require('./commands/doctor');
const { registerFormatCommand } = require('./commands/format');

// 配置主程序
program
  .name('wxmd-cli')
  .description('微信公众号 Markdown 编辑器 CLI')
  .version(pkg.version, '-v, --version', 'Display version number')
  .helpOption('-h, --help', 'Display help for command')
  .configureHelp({
    sortSubcommands: true,
    subcommandTerm: (cmd) => cmd.name(),
  });

// 全局选项
program
  .option('-o, --output <format>', 'Output format: json, html, text', 'json')
  .option('--trace-id <id>', 'Trace ID for request tracking')
  .option('--timeout <ms>', 'Request timeout in milliseconds', '30000');

// 注册命令
registerTypesetCommand(program);
registerShareCommand(program);
registerStylesCommand(program);
registerDoctorCommand(program);
registerFormatCommand(program);

// 自定义帮助信息
program.on('--help', () => {
  console.log('');
  console.log('Examples:');
  console.log('  $ wxmd-cli typeset --input article.md --style wechat-tech');
  console.log('  $ echo "# Hello" | wxmd-cli typeset --output html');
  console.log('  $ wxmd-cli share create --input article.md --style wechat-default');
  console.log('  $ wxmd-cli share get abc12345');
  console.log('  $ wxmd-cli styles list');
  console.log('  $ wxmd-cli doctor');
  console.log('  $ wxmd-cli format --input article.md --out fixed.md');
  console.log('  $ echo "hello世界" | wxmd-cli format');
  console.log('');
  console.log('Environment Variables:');
  console.log('  WXMD_API_URL    API server URL (default: http://localhost:8080)');
  console.log('  WXMD_API_TIMEOUT  Request timeout in ms (default: 30000)');
});

// 解析参数
program.parse();

// 如果没有参数，显示帮助
if (process.argv.length <= 2) {
  program.help();
}
