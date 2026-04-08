/**
 * share 命令
 * 分享相关操作：create, get
 */

const fs = require('fs');
const { createShare, getShare, setGlobalConfig } = require('../lib/api');
const { success, error, formatOutput, getExitCode } = require('../lib/output');
const { hasStyle, getDefaultStyle, resolveStyleKey } = require('../lib/styles');

/**
 * 从文件或 stdin 读取内容
 * @param {string|null} filePath
 * @returns {Promise<string>}
 */
async function readInput(filePath) {
  if (filePath) {
    if (!fs.existsSync(filePath)) {
      throw new Error(`File not found: ${filePath}`);
    }
    return fs.readFileSync(filePath, 'utf-8');
  } else {
    return new Promise((resolve, reject) => {
      let data = '';
      process.stdin.setEncoding('utf8');

      process.stdin.on('data', (chunk) => {
        data += chunk;
      });

      process.stdin.on('end', () => {
        resolve(data);
      });

      process.stdin.on('error', (err) => {
        reject(err);
      });

      if (process.stdin.isTTY) {
        resolve('');
      }
    });
  }
}

/**
 * create 子命令
 */
async function shareCreateAction(options) {
  try {
    const { input, style, output, traceId, timeout } = options;
    const resolvedStyle = resolveStyleKey(style);

    // 设置 API 全局配置
    setGlobalConfig({ traceId, timeout });

    // 验证样式
    if (!hasStyle(style)) {
      const result = error(
        'INVALID_STYLE',
        'INVALID_ARGUMENTS',
        `Style '${style}' not found`,
        {
          actionHint: `Run 'wxmd-cli styles list' to see available styles`,
        }
      );
      console.log(formatOutput(result, output));
      process.exit(getExitCode(result));
      return;
    }

    // 读取内容
    let content;
    try {
      content = await readInput(input);
    } catch (err) {
      const result = error(
        'READ_ERROR',
        'INVALID_ARGUMENTS',
        err.message,
        {
          actionHint: 'Check if the file exists and is readable',
        }
      );
      console.log(formatOutput(result, output));
      process.exit(getExitCode(result));
      return;
    }

    if (!content || !content.trim()) {
      const result = error(
        'EMPTY_CONTENT',
        'INVALID_ARGUMENTS',
        'No content provided',
        {
          actionHint: 'Provide content via --input file.md or stdin',
        }
      );
      console.log(formatOutput(result, output));
      process.exit(getExitCode(result));
      return;
    }

    // 创建分享
    const response = await createShare({ content, style: resolvedStyle });

    if (response.success) {
      const data = response.data;
      const result = success(
        {
          id: data.id,
          url: `${process.env.WXMD_API_URL || 'http://localhost:8080'}/s/${data.id}`,
          createdAt: data.createdAt,
          style: data.style,
        },
        { apiStatus: response.status }
      );
      console.log(formatOutput(result, output));
      process.exit(getExitCode(result));
    } else {
      const result = error(
        'CREATE_FAILED',
        'SERVER_ERROR',
        response.error || 'Failed to create share',
        {
          retryable: true,
          actionHint: 'Check server status and try again',
        }
      );
      console.log(formatOutput(result, output));
      process.exit(getExitCode(result));
    }

  } catch (err) {
    const isNetworkError = err.code === 'ECONNREFUSED' || err.code === 'ENOTFOUND';
    const result = error(
      isNetworkError ? 'NETWORK_ERROR' : 'CREATE_ERROR',
      isNetworkError ? 'NETWORK_ERROR' : 'GENERAL_ERROR',
      err.message || 'Failed to create share',
      {
        retryable: isNetworkError,
        actionHint: isNetworkError
          ? 'Check if the server is running and API_URL is correct'
          : 'Check error details and try again',
        details: err.code,
      }
    );
    console.log(formatOutput(result, options.output));
    process.exit(getExitCode(result));
  }
}

/**
 * get 子命令
 */
async function shareGetAction(shareId, options) {
  try {
    const { output, traceId, timeout } = options;

    // 设置 API 全局配置
    setGlobalConfig({ traceId, timeout });

    if (!shareId) {
      const result = error(
        'MISSING_ID',
        'INVALID_ARGUMENTS',
        'Share ID is required',
        {
          actionHint: 'Use: wxmd-cli share get <id>',
        }
      );
      console.log(formatOutput(result, output));
      process.exit(getExitCode(result));
      return;
    }

    const response = await getShare(shareId);

    if (response.success) {
      const data = response.data;
      const result = success(
        {
          id: data.id,
          content: data.content,
          style: data.style,
          createdAt: data.createdAt,
          updatedAt: data.updatedAt,
        },
        { apiStatus: response.status }
      );
      console.log(formatOutput(result, output));
      process.exit(getExitCode(result));
    } else {
      const isNotFound = response.status === 404;
      const result = error(
        isNotFound ? 'NOT_FOUND' : 'GET_FAILED',
        isNotFound ? 'NOT_FOUND' : 'SERVER_ERROR',
        response.error || (isNotFound ? 'Share not found' : 'Failed to get share'),
        {
          retryable: !isNotFound,
          actionHint: isNotFound
            ? 'Check the share ID and try again'
            : 'Check server status and try again',
        }
      );
      console.log(formatOutput(result, output));
      process.exit(getExitCode(result));
    }

  } catch (err) {
    const isNetworkError = err.code === 'ECONNREFUSED' || err.code === 'ENOTFOUND';
    const result = error(
      isNetworkError ? 'NETWORK_ERROR' : 'GET_ERROR',
      isNetworkError ? 'NETWORK_ERROR' : 'GENERAL_ERROR',
      err.message || 'Failed to get share',
      {
        retryable: isNetworkError,
        actionHint: isNetworkError
          ? 'Check if the server is running and API_URL is correct'
          : 'Check error details and try again',
        details: err.code,
      }
    );
    console.log(formatOutput(result, options.output));
    process.exit(getExitCode(result));
  }
}

/**
 * 注册 share 命令
 */
function registerShareCommand(parentProgram) {
  const shareCmd = parentProgram
    .command('share')
    .description('Share operations - create and get shares');

  // create 子命令
  shareCmd
    .command('create')
    .description('Create a new share')
    .option('-i, --input <file>', 'Input markdown file (default: stdin)')
    .option('-s, --style <name>', 'Style theme', getDefaultStyle())
    .action((options, command) => {
      const globalOpts = command.optsWithGlobals();
      options.output = globalOpts.output || options.output;
      options.traceId = globalOpts.traceId;
      options.timeout = globalOpts.timeout;
      return shareCreateAction(options);
    });

  // get 子命令
  shareCmd
    .command('get <id>')
    .description('Get a share by ID')
    .action((shareId, options, command) => {
      const globalOpts = command.optsWithGlobals();
      options.output = globalOpts.output || options.output;
      options.traceId = globalOpts.traceId;
      options.timeout = globalOpts.timeout;
      return shareGetAction(shareId, options);
    });
}

module.exports = {
  registerShareCommand,
};
