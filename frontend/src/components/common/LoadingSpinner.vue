<template>
  <div 
    class="loading-spinner" 
    :class="{ 'full-screen': fullScreen, [`size-${size}`]: true }"
  >
    <div v-if="fullScreen" class="loading-overlay">
      <div class="loading-content">
        <div class="spinner-container">
          <el-icon v-if="type === 'default'" class="spinner default-spinner">
            <Loading />
          </el-icon>
          
          <div v-else-if="type === 'dots'" class="dots-spinner">
            <span class="dot"></span>
            <span class="dot"></span>
            <span class="dot"></span>
          </div>
          
          <div v-else-if="type === 'bars'" class="bars-spinner">
            <span class="bar"></span>
            <span class="bar"></span>
            <span class="bar"></span>
            <span class="bar"></span>
            <span class="bar"></span>
          </div>
          
          <div v-else-if="type === 'circle'" class="circle-spinner">
            <div class="circle"></div>
          </div>
          
          <div v-else-if="type === 'wave'" class="wave-spinner">
            <span class="wave"></span>
            <span class="wave"></span>
            <span class="wave"></span>
            <span class="wave"></span>
            <span class="wave"></span>
          </div>
        </div>
        
        <div v-if="text" class="loading-text">{{ text }}</div>
        <div v-if="description" class="loading-description">{{ description }}</div>
        
        <!-- 进度条 -->
        <div v-if="showProgress && progress >= 0" class="loading-progress">
          <el-progress
            :percentage="progress"
            :show-text="false"
            :stroke-width="4"
          />
          <div class="progress-text">{{ progress }}%</div>
        </div>
      </div>
    </div>
    
    <!-- 非全屏模式 -->
    <template v-else>
      <div class="spinner-container">
        <el-icon v-if="type === 'default'" class="spinner default-spinner">
          <Loading />
        </el-icon>
        
        <div v-else-if="type === 'dots'" class="dots-spinner">
          <span class="dot"></span>
          <span class="dot"></span>
          <span class="dot"></span>
        </div>
        
        <div v-else-if="type === 'bars'" class="bars-spinner">
          <span class="bar"></span>
          <span class="bar"></span>
          <span class="bar"></span>
          <span class="bar"></span>
          <span class="bar"></span>
        </div>
        
        <div v-else-if="type === 'circle'" class="circle-spinner">
          <div class="circle"></div>
        </div>
        
        <div v-else-if="type === 'wave'" class="wave-spinner">
          <span class="wave"></span>
          <span class="wave"></span>
          <span class="wave"></span>
          <span class="wave"></span>
          <span class="wave"></span>
        </div>
      </div>
      
      <div v-if="text" class="loading-text">{{ text }}</div>
      <div v-if="description" class="loading-description">{{ description }}</div>
    </template>
  </div>
</template>

<script setup>
import { Loading } from '@element-plus/icons-vue'

defineProps({
  // 加载器类型
  type: {
    type: String,
    default: 'default',
    validator: (value) => ['default', 'dots', 'bars', 'circle', 'wave'].includes(value)
  },
  // 大小
  size: {
    type: String,
    default: 'medium',
    validator: (value) => ['small', 'medium', 'large'].includes(value)
  },
  // 加载文本
  text: {
    type: String,
    default: ''
  },
  // 描述文本
  description: {
    type: String,
    default: ''
  },
  // 是否全屏
  fullScreen: {
    type: Boolean,
    default: false
  },
  // 是否显示进度
  showProgress: {
    type: Boolean,
    default: false
  },
  // 进度值（0-100）
  progress: {
    type: Number,
    default: -1
  },
  // 主题颜色
  color: {
    type: String,
    default: '#409eff'
  }
})
</script>

<style scoped>
.loading-spinner {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: v-bind(color);
}

.full-screen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 9999;
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.spinner-container {
  margin-bottom: 16px;
}

.size-small .spinner-container {
  font-size: 20px;
}

.size-medium .spinner-container {
  font-size: 24px;
}

.size-large .spinner-container {
  font-size: 32px;
}

/* 默认加载器 */
.default-spinner {
  animation: rotate 1s linear infinite;
}

/* 点状加载器 */
.dots-spinner {
  display: inline-flex;
  gap: 4px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
  animation: dot-bounce 1.4s ease-in-out infinite both;
}

.size-small .dot {
  width: 6px;
  height: 6px;
}

.size-large .dot {
  width: 10px;
  height: 10px;
}

.dot:nth-child(1) { animation-delay: -0.32s; }
.dot:nth-child(2) { animation-delay: -0.16s; }
.dot:nth-child(3) { animation-delay: 0s; }

/* 条状加载器 */
.bars-spinner {
  display: inline-flex;
  gap: 3px;
  align-items: end;
}

.bar {
  width: 4px;
  height: 20px;
  background: currentColor;
  animation: bar-scale 1.2s ease-in-out infinite;
}

.size-small .bar {
  width: 3px;
  height: 16px;
}

.size-large .bar {
  width: 5px;
  height: 28px;
}

.bar:nth-child(1) { animation-delay: 0s; }
.bar:nth-child(2) { animation-delay: 0.1s; }
.bar:nth-child(3) { animation-delay: 0.2s; }
.bar:nth-child(4) { animation-delay: 0.3s; }
.bar:nth-child(5) { animation-delay: 0.4s; }

/* 圆圈加载器 */
.circle-spinner {
  position: relative;
  display: inline-block;
}

.circle {
  width: 32px;
  height: 32px;
  border: 3px solid transparent;
  border-top: 3px solid currentColor;
  border-radius: 50%;
  animation: circle-rotate 1s linear infinite;
}

.size-small .circle {
  width: 24px;
  height: 24px;
  border-width: 2px;
  border-top-width: 2px;
}

.size-large .circle {
  width: 40px;
  height: 40px;
  border-width: 4px;
  border-top-width: 4px;
}

/* 波浪加载器 */
.wave-spinner {
  display: inline-flex;
  gap: 2px;
  align-items: center;
}

.wave {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  animation: wave-scale 1.5s ease-in-out infinite;
}

.size-small .wave {
  width: 4px;
  height: 4px;
}

.size-large .wave {
  width: 8px;
  height: 8px;
}

.wave:nth-child(1) { animation-delay: 0s; }
.wave:nth-child(2) { animation-delay: 0.1s; }
.wave:nth-child(3) { animation-delay: 0.2s; }
.wave:nth-child(4) { animation-delay: 0.3s; }
.wave:nth-child(5) { animation-delay: 0.4s; }

/* 文本样式 */
.loading-text {
  font-size: 14px;
  font-weight: 500;
  color: #333;
  margin-bottom: 8px;
}

.loading-description {
  font-size: 12px;
  color: #666;
  margin-bottom: 16px;
}

.loading-progress {
  width: 200px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.progress-text {
  font-size: 12px;
  color: #666;
}

/* 动画定义 */
@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@keyframes dot-bounce {
  0%, 80%, 100% {
    transform: scale(0);
    opacity: 0.5;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}

@keyframes bar-scale {
  0%, 40%, 100% {
    transform: scaleY(0.4);
  }
  20% {
    transform: scaleY(1);
  }
}

@keyframes circle-rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@keyframes wave-scale {
  0%, 60%, 100% {
    transform: scale(1);
    opacity: 1;
  }
  30% {
    transform: scale(1.5);
    opacity: 0.7;
  }
}
</style>