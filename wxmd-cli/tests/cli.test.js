/**
 * CLI 测试套件
 */

const { describe, it } = require('node:test');
const assert = require('node:assert');
const frontendAutocorrect = require('../../frontend/lib/autocorrect.js');
const { renderMarkdown } = require('../src/lib/renderer');
const { success, error, getExitCode } = require('../src/lib/output');
const { listStyles, hasStyle, getStyle } = require('../src/lib/styles');

describe('styles', () => {
  it('should list available styles', () => {
    const styles = listStyles();
    assert.strictEqual(Array.isArray(styles), true);
    assert.strictEqual(styles.length > 0, true);
    assert.ok(styles.some(s => s.key === 'wechat-default'));
  });

  it('should check if style exists', () => {
    assert.strictEqual(hasStyle('wechat-default'), true);
    assert.strictEqual(hasStyle('nonexistent'), false);
  });

  it('should get style config', () => {
    const style = getStyle('wechat-default');
    assert.ok(style);
    assert.ok(style.name);
    assert.ok(style.styles);
    assert.ok(style.styles.container);
  });
});

describe('renderer', () => {
  it('should render markdown to HTML', () => {
    const markdown = '# Hello\n\nWorld';
    const html = renderMarkdown(markdown, 'wechat-default');
    assert.ok(html.includes('<h1'));
    assert.ok(html.includes('Hello'));
    assert.ok(html.includes('<p'));
    assert.ok(html.includes('World'));
  });

  it('should apply correct style', () => {
    const markdown = '# Test';
    const html = renderMarkdown(markdown, 'wechat-tech');
    assert.ok(html.includes('border-bottom: 3px solid #0066cc'));
  });

  it('should render lists', () => {
    const markdown = '- item 1\n- item 2';
    const html = renderMarkdown(markdown, 'wechat-default');
    assert.ok(html.includes('<ul'));
    assert.ok(html.includes('<li'));
  });

  it('should render code blocks', () => {
    const markdown = '```js\nconst x = 1;\n```';
    const html = renderMarkdown(markdown, 'wechat-default');
    assert.ok(html.includes('<pre'));
    assert.ok(html.includes('<code'));
  });

  it('should strip citation markers', () => {
    const markdown = 'Hello \uE200cite\uE202turn1search1\uE201 World';
    const html = renderMarkdown(markdown, 'wechat-default');
    assert.ok(!html.includes('\uE200'));
  });
});

describe('autocorrect', () => {
  it('should preserve markdown image syntax when fixing spaces', () => {
    const markdown = '![Nemotron 3 Ultra 架构示意] => ![Nemotron 3 Ultra 架构示意]';
    const fixed = frontendAutocorrect.format(markdown);
    assert.strictEqual(fixed, markdown);
  });
});

describe('output', () => {
  it('should create success response', () => {
    const result = success({ test: 'data' });
    assert.strictEqual(result.ok, true);
    assert.strictEqual(result.data.test, 'data');
    assert.ok(result.meta.cliVersion);
    assert.ok(result.meta.timestamp);
  });

  it('should create error response', () => {
    const result = error('CODE', 'TYPE', 'message');
    assert.strictEqual(result.ok, false);
    assert.strictEqual(result.error.code, 'CODE');
    assert.strictEqual(result.error.type, 'TYPE');
    assert.strictEqual(result.error.message, 'message');
  });

  it('should return correct exit codes', () => {
    const successResult = success({});
    assert.strictEqual(getExitCode(successResult), 0);

    const errorResult = error('CODE', 'NETWORK_ERROR', 'msg');
    assert.strictEqual(getExitCode(errorResult), 3);

    const notFoundResult = error('CODE', 'NOT_FOUND', 'msg');
    assert.strictEqual(getExitCode(notFoundResult), 5);
  });
});
