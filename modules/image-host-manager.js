/**
 * 图床管理器 - 支持多个图床服务，智能降级
 */
(() => {
class ImageHostManager {
  constructor() {
    // 图床服务列表（仅保留可靠且无CORS限制的服务）
    this.hosts = [
      {
        name: 'SM.MS',
        upload: this.uploadToSmms.bind(this),
        maxSize: 5 * 1024 * 1024, // 5MB
        priority: 1,
        timeout: 10000 // 10秒超时
      }
    ];

    // 失败记录（用于临时降低优先级）
    this.failureCount = {};
    this.lastFailureTime = {};

    // 启用/禁用状态（可以手动禁用某些服务）
    this.disabledHosts = new Set();
  }

  // 智能选择图床（根据失败记录和文件大小）
  selectHost(fileSize) {
    const now = Date.now();
    const cooldownTime = 3 * 60 * 1000; // 3分钟冷却时间（缩短以便更快重试）

    return this.hosts
      .filter(host => {
        // 过滤条件：1) 文件大小符合 2) 未被禁用 3) 不在冷却期或失败次数不太多
        if (fileSize > host.maxSize) return false;
        if (this.disabledHosts.has(host.name)) return false;

        const failures = this.failureCount[host.name] || 0;
        const lastFail = this.lastFailureTime[host.name] || 0;
        const inCooldown = (now - lastFail) < cooldownTime;

        // 如果失败次数超过3次且在冷却期内，跳过
        if (failures >= 3 && inCooldown) return false;

        return true;
      })
      .sort((a, b) => {
        // 如果最近失败过，降低优先级
        const aFailures = this.failureCount[a.name] || 0;
        const bFailures = this.failureCount[b.name] || 0;
        const aLastFail = this.lastFailureTime[a.name] || 0;
        const bLastFail = this.lastFailureTime[b.name] || 0;

        // 如果在冷却期内，大幅降低优先级
        const aInCooldown = (now - aLastFail) < cooldownTime;
        const bInCooldown = (now - bLastFail) < cooldownTime;

        if (aInCooldown && !bInCooldown) return 1;
        if (!aInCooldown && bInCooldown) return -1;

        // 按失败次数和原始优先级排序
        const aPenalty = aFailures * 5 + a.priority;
        const bPenalty = bFailures * 5 + b.priority;

        return aPenalty - bPenalty;
      });
  }

  // 记录失败
  recordFailure(hostName) {
    this.failureCount[hostName] = (this.failureCount[hostName] || 0) + 1;
    this.lastFailureTime[hostName] = Date.now();
  }

  // 记录成功（重置失败计数）
  recordSuccess(hostName) {
    this.failureCount[hostName] = 0;
    delete this.lastFailureTime[hostName];
  }

  // 尝试上传到所有可用图床
  async upload(file, onProgress) {
    const availableHosts = this.selectHost(file.size);

    if (availableHosts.length === 0) {
      throw new Error('没有可用的图床服务（文件可能太大或所有服务都在冷却期）');
    }

    let lastError = null;
    let attemptCount = 0;

    for (const host of availableHosts) {
      attemptCount++;
      try {
        if (onProgress) {
          onProgress(`🔄 尝试 ${host.name} (${attemptCount}/${availableHosts.length})`);
        }

        // 使用Promise.race实现超时控制
        const uploadPromise = host.upload(file);
        const timeoutPromise = new Promise((_, reject) => {
          setTimeout(() => reject(new Error('上传超时')), host.timeout);
        });

        const result = await Promise.race([uploadPromise, timeoutPromise]);
        this.recordSuccess(host.name);

        if (onProgress) {
          onProgress(`✅ ${host.name} 上传成功`);
        }

        return {
          url: result.url,
          host: host.name,
          deleteUrl: result.deleteUrl
        };
      } catch (error) {
        const errorMsg = error.message || error.toString();
        console.warn(`${host.name} 上传失败:`, errorMsg);
        this.recordFailure(host.name);
        lastError = error;

        // 如果还有其他图床可以尝试，继续
        if (attemptCount < availableHosts.length && onProgress) {
          onProgress(`⚠️ ${host.name} 失败，尝试下一个...`);
        }
      }
    }

    // 所有图床都失败了
    throw new Error(`所有图床均上传失败 (尝试了${attemptCount}个)\n最后错误: ${lastError?.message || '未知错误'}`);
  }

  // SM.MS 图床（唯一支持浏览器端直接上传的稳定图床）
  async uploadToSmms(file) {
    const formData = new FormData();
    formData.append('smfile', file);

    const response = await fetch('https://sm.ms/api/v2/upload', {
      method: 'POST',
      body: formData
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const result = await response.json();

    if (result.success || (result.code === 'image_repeated' && result.images)) {
      return {
        url: result.data?.url || result.images,
        deleteUrl: result.data?.delete || null
      };
    }

    throw new Error(result.message || 'SM.MS响应失败');
  }

  // 辅助：文件转 Base64
  fileToBase64(file) {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.readAsDataURL(file);
      reader.onload = () => resolve(reader.result);
      reader.onerror = error => reject(error);
    });
  }
}

  window.WechatEditorModules = window.WechatEditorModules || {};
  window.WechatEditorModules.ImageHostManager = ImageHostManager;
})();
