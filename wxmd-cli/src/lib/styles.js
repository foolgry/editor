const fs = require('fs');
const path = require('path');
const vm = require('vm');

const FRONTEND_STYLES_PATH = path.resolve(__dirname, '../../../frontend/styles.js');
const FALLBACK_DEFAULT_STYLE = 'wechat-anthropic';

let cachedStyles = null;
let cachedStyleAliasMap = null;

function normalizeStyleAlias(rawValue) {
  return String(rawValue || '')
    .trim()
    .toLowerCase()
    .replace(/[()]/g, '')
    .replace(/[（）]/g, '')
    .replace(/隐藏/g, '')
    .replace(/\s+/g, '-')
    .replace(/_+/g, '-')
    .replace(/-+/g, '-');
}

function loadStylesFromFrontend() {
  if (!fs.existsSync(FRONTEND_STYLES_PATH)) {
    throw new Error(`Frontend styles not found: ${FRONTEND_STYLES_PATH}`);
  }

  const source = fs.readFileSync(FRONTEND_STYLES_PATH, 'utf8');
  const sandbox = {
    console,
    window: {
      WechatEditorModules: {},
    },
  };
  sandbox.window.window = sandbox.window;

  vm.runInNewContext(source, sandbox, {
    filename: FRONTEND_STYLES_PATH,
  });

  const styles = sandbox.window.WechatEditorModules.STYLES;
  if (!styles || typeof styles !== 'object') {
    throw new Error(`Failed to load STYLES from ${FRONTEND_STYLES_PATH}`);
  }

  return styles;
}

function getStyles() {
  if (!cachedStyles) {
    cachedStyles = loadStylesFromFrontend();
  }
  return cachedStyles;
}

function createStyleAliasMap(styles) {
  const aliases = {
    claude: 'wechat-anthropic',
    anthropic: 'wechat-anthropic',
    'claude-song': 'wechat-claude-song',
    'claude-song-serif': 'wechat-claude-song',
    default: 'wechat-default',
  };

  Object.entries(styles).forEach(([styleKey, styleConfig]) => {
    const normalizedKey = normalizeStyleAlias(styleKey);
    if (normalizedKey) {
      aliases[normalizedKey] = styleKey;
    }

    const normalizedName = normalizeStyleAlias(styleConfig && styleConfig.name);
    if (normalizedName && !aliases[normalizedName]) {
      aliases[normalizedName] = styleKey;
    }
  });

  return aliases;
}

function getStyleAliasMap() {
  if (!cachedStyleAliasMap) {
    cachedStyleAliasMap = createStyleAliasMap(getStyles());
  }
  return cachedStyleAliasMap;
}

function findResolvedStyleKey(styleKey) {
  const styles = getStyles();

  if (styleKey && styles[styleKey]) {
    return styleKey;
  }

  const normalizedStyleKey = normalizeStyleAlias(styleKey);
  if (!normalizedStyleKey) {
    return null;
  }

  const resolvedStyleKey = getStyleAliasMap()[normalizedStyleKey];
  if (resolvedStyleKey && styles[resolvedStyleKey]) {
    return resolvedStyleKey;
  }

  return null;
}

function resolveStyleKey(styleKey) {
  const styles = getStyles();
  const resolvedStyleKey = findResolvedStyleKey(styleKey);
  if (resolvedStyleKey) {
    return resolvedStyleKey;
  }

  if (styles[FALLBACK_DEFAULT_STYLE]) {
    return FALLBACK_DEFAULT_STYLE;
  }

  if (styles['wechat-default']) {
    return 'wechat-default';
  }

  const [firstStyleKey] = Object.keys(styles);
  return firstStyleKey || null;
}

function listStyles() {
  return Object.entries(getStyles()).map(([key, config]) => ({
    key,
    name: String(config.name || key)
      .replace(/（隐藏）/g, '')
      .replace(/\(隐藏\)/g, '')
      .trim(),
    hidden: Boolean(config.hidden),
  }));
}

function getStyle(styleKey) {
  const resolvedStyleKey = styleKey ? findResolvedStyleKey(styleKey) : getDefaultStyle();
  return resolvedStyleKey ? getStyles()[resolvedStyleKey] || null : null;
}

function hasStyle(styleKey) {
  return Boolean(findResolvedStyleKey(styleKey));
}

function getDefaultStyle() {
  return resolveStyleKey(FALLBACK_DEFAULT_STYLE);
}

module.exports = {
  FRONTEND_STYLES_PATH,
  STYLES: getStyles(),
  normalizeStyleAlias,
  resolveStyleKey,
  listStyles,
  getStyle,
  hasStyle,
  getDefaultStyle,
};
