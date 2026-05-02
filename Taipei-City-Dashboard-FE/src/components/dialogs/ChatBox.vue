<script setup>
import { ref, watch, nextTick, computed } from "vue";
import { storeToRefs } from "pinia";
import SendIcon from "../icons/SendIcon.vue";
import BotLogo from "../icons/BotLogo.vue";
import UserLogo from "../icons/UserLogo.vue";

import { useChatStore } from "../../store/chatStore";
import { useContentStore } from "../../store/contentStore";
import { useAuthStore } from "../../store/authStore";
import http from "../../router/axios";

const chatStore = useChatStore();
const contentStore = useContentStore();
const authStore = useAuthStore();
const { addChatData, addQueryData, addGenerativeQuery, saveChatLog, toggleMode, toggleComponent, selectAllComponents, deselectAllComponents } = chatStore;
const { createDashboard } = contentStore;
const { chatData, chatMode, selectedComponentIds, isGenerating } = storeToRefs(chatStore);
const { editDashboard, currentDashboard } = storeToRefs(contentStore);
const { user } = storeToRefs(authStore);

const userMessage = ref("");
const chatAreaRef = ref(null);
const isStickyOpen = ref(false);
const dashboardCreationLoading = ref(false);
const showComponentPicker = ref(false);

// 當前儀表板的組件列表
const currentComponents = computed(() => {
	if (!currentDashboard.value?.components) return [];
	return currentDashboard.value.components.filter(c => c && c.name);
});

// 全選狀態
const isAllSelected = computed(() => {
	if (currentComponents.value.length === 0) return false;
	return currentComponents.value.every(c => selectedComponentIds.value.includes(c.id));
});

const qaBtnHandler = async (text, relations) => {
	if (text === "建立儀表板") {
		if (dashboardCreationLoading.value === true) return;
		dashboardCreationLoading.value = true;
		const response = await http.get(`/dashboard/`);
		if (response.data?.data?.personal?.length > 20) {
			addChatData({
				role: "bot",
				content: "您的個人儀表板已超出限制 20 個，請先移除既有儀表板後，重新執行本功能！",
			});
			dashboardCreationLoading.value = false;
			return;
		}
		const components = Array.from(new Set(relations.map((r) => r.id))).map(
			(id) => ({ id }),
		);

		if (user.value.user_id) {
			editDashboard.value = {
				index: "",
				name: "推薦儀表板",
				icon: "star",
				components: components,
			};
			await createDashboard();
			saveChatLog("建立儀表板", "使用者成功建立儀表板!");
		} else {
			addChatData({
				role: "bot",
				content: "請先登入會員以使用此功能喔！",
			});
		}
		dashboardCreationLoading.value = false;
	}
};

const sendBtnHandler = (text) => {
	if (!text.trim()) return;
	if (isGenerating.value) return;

	if (chatMode.value === 'generative') {
		// 生成式問答模式
		addGenerativeQuery(
			{ role: "user", content: text },
			currentComponents.value
		);
	} else {
		// 搜尋模式
		addQueryData({ role: "user", content: text });
	}
	userMessage.value = "";
};

const toggleSticky = () => {
	isStickyOpen.value = !isStickyOpen.value;
};

const handleToggleAll = () => {
	if (isAllSelected.value) {
		deselectAllComponents();
	} else {
		selectAllComponents(currentComponents.value.map(c => c.id));
	}
};

watch(
	() => chatData.value.length,
	async () => {
		await nextTick();
		const chat = chatAreaRef.value;
		if (!chat) return;
		chat.scrollTop = chat.scrollHeight - chat.clientHeight;
	},
	{ deep: true },
);

// 切換到生成式模式時，預設全選組件
watch(
	() => chatMode.value,
	(newMode) => {
		if (newMode === 'generative' && currentComponents.value.length > 0) {
			selectAllComponents(currentComponents.value.map(c => c.id));
		}
	}
);
</script>

<template>
  <div class="chat-widget">
    <!-- 標題 -->
    <div class="header">
      <h3>臺北城市儀表板小幫手</h3>
      <!-- 模式切換 -->
      <div class="mode-toggle">
        <button
          :class="{ active: chatMode === 'search' }"
          @click="toggleMode('search')"
        >
          組件搜尋
        </button>
        <button
          :class="{ active: chatMode === 'generative' }"
          @click="toggleMode('generative')"
        >
          數據問答
        </button>
      </div>
    </div>

    <!-- 生成式模式：組件勾選區 -->
    <div
      v-if="chatMode === 'generative'"
      class="component-picker"
    >
      <div
        class="component-picker-header"
        @click="showComponentPicker = !showComponentPicker"
      >
        <span>參考組件 ({{ selectedComponentIds.length }}/{{ currentComponents.length }})</span>
        <button class="toggle-btn">
          {{ showComponentPicker ? '▲' : '▼' }}
        </button>
      </div>
      <div
        v-show="showComponentPicker"
        class="component-picker-list"
      >
        <label class="component-picker-item select-all">
          <input
            type="checkbox"
            :checked="isAllSelected"
            @change="handleToggleAll"
          >
          <span>全選</span>
        </label>
        <label
          v-for="comp in currentComponents"
          :key="comp.id"
          class="component-picker-item"
        >
          <input
            type="checkbox"
            :checked="selectedComponentIds.includes(comp.id)"
            @change="toggleComponent(comp.id)"
          >
          <span>{{ comp.name }}</span>
        </label>
      </div>
    </div>

    <!-- 聊天區 -->
    <div
      ref="chatAreaRef"
      class="chat-area scrollbar-custom"
    >
      <!-- 置頂訊息 -->
      <div class="chat-message sticky-message">
        <div
          class="sticky-header"
          @click="toggleSticky"
        >
          <span>置頂公告：小幫手使用須知</span>
          <button class="toggle-btn">
            {{ isStickyOpen ? "-" : "+" }}
          </button>
        </div>
        <div
          v-show="isStickyOpen"
          class="sticky-body"
        >
          <span v-if="chatMode === 'search'">小幫手會依據您輸入的內容，自動檢索本站臺的組件資料庫，並回傳相似度較高的組件清單，協助您快速找到符合需求的元件或資訊。<br><br>
            目前搜尋模式僅提供組件比對與分析服務。切換至「數據問答」模式可直接向 AI 提問。</span>
          <span v-else>在「數據問答」模式下，您可以勾選當前儀表板的組件，AI 會根據這些組件的實際數據回答您的問題。<br><br>
            例如：「台北市太陽能裝置容量近五年成長了多少？」「哪個行政區的日照量最高？」</span>
        </div>
      </div>
      <div
        v-for="chat in chatData"
        :key="chat.id"
        class="message"
      >
        <!-- 機器人訊息 -->
        <div
          v-if="chat.role === 'bot'"
          class="bot"
        >
          <div class="avatar">
            <BotLogo />
          </div>
          <div class="content">
            <div
              v-if="chat.content"
              class="message--bubble"
              :class="{ 'is-loading': chat.isLoading, 'is-generative': chat.isGenerative }"
            >
              <p>{{ chat.content }}</p>
            </div>
            <!-- 表格區 -->
            <div
              v-if="chat.relations"
              v-horizontal-wheel
              class="relation-area"
            >
              <table class="relation-table">
                <thead>
                  <tr>
                    <th>排名</th>
                    <th>城市名</th>
                    <th>組件名</th>
                    <th>關聯性</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="(item, index) in chat.relations"
                    :key="index"
                  >
                    <td>{{ index + 1 }}</td>
                    <td>
                      {{
                        item.city === "taipei"
                          ? "臺北"
                          : "雙北"
                      }}
                    </td>
                    <td>{{ item.name }}</td>
                    <td>{{ item.score }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div
              v-if="chat.button"
              v-horizontal-wheel
              class="message--button scrollbar-x-hide"
            >
              <button
                v-for="btn in chat.button"
                :key="btn.id"
                @click="qaBtnHandler(btn.text, chat.relations)"
              >
                {{ btn.text }}
              </button>
            </div>
          </div>
        </div>
        <!-- 使用者訊息 -->
        <div
          v-else
          class="user"
        >
          <div class="avatar">
            <UserLogo />
          </div>
          <div
            v-if="chat.content"
            class="content"
          >
            <div class="message--bubble">
              <p>{{ chat.content }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 輸入區 -->
    <div class="input-area">
      <input
        v-model="userMessage"
        type="text"
        :placeholder="chatMode === 'generative' ? '根據勾選的組件數據提問...' : '輸入訊息...'"
        :disabled="isGenerating"
        @keyup.enter="sendBtnHandler(userMessage)"
      >
      <button
        :disabled="isGenerating"
        @click="sendBtnHandler(userMessage)"
      >
        <SendIcon />
      </button>
    </div>
  </div>
</template>

<style lang="scss" scoped>
/* === 變數設定 === */
$bg-dark: #090909;
$panel-bg: #494b4e;
$card-bg: #282a2c;
$border-color: #888787;
$input-bg: #d9d9d9;
$white: #ffffff;
$scroll-thumb-hover: #ababab;
$radius-10: 10px;
$radius-15: 15px;
$radius-20: 20px;

/* === Scrollbar === */
.scrollbar-x-hide {
	scrollbar-width: none;

	&::-webkit-scrollbar {
		display: none;
	}
}

.scrollbar-custom {
	&::-webkit-scrollbar {
		width: 2px;
		background: transparent;
	}

	&::-webkit-scrollbar-thumb {
		background: $white;
		border-radius: 8px;
	}

	&::-webkit-scrollbar-thumb:hover {
		background: $scroll-thumb-hover;
	}
}

/* === 主要樣式 === */
.chat-widget {
	width: 400px;
	border-radius: $radius-20;
	overflow: hidden;
	background: $bg-dark;
	border: 1px solid $border-color;
	display: flex;
	flex-direction: column;

	.header {
		padding: 0.8rem 1rem;
		background: $panel-bg;
		border-bottom: 3px solid $border-color;

		h3 {
			font-size: 16px;
			font-weight: 700;
			color: $white;
			margin: 0 0 0.5rem 0;
		}
	}

	.mode-toggle {
		display: flex;
		gap: 4px;
		background: rgba(0, 0, 0, 0.3);
		border-radius: 8px;
		padding: 2px;

		button {
			flex: 1;
			padding: 5px 12px;
			border: none;
			border-radius: 6px;
			background: transparent;
			color: #999;
			font-size: 13px;
			cursor: pointer;
			transition: all 0.2s;

			&.active {
				background: #5b9bd5;
				color: $white;
				font-weight: 600;
			}

			&:hover:not(.active) {
				color: $white;
			}
		}
	}

	.component-picker {
		border-bottom: 1px solid $border-color;
		background: #1a1a1a;

		&-header {
			display: flex;
			justify-content: space-between;
			align-items: center;
			padding: 8px 12px;
			cursor: pointer;
			font-size: 13px;
			color: #ccc;

			&:hover {
				color: $white;
			}

			.toggle-btn {
				background: none;
				border: none;
				color: inherit;
				font-size: 10px;
				cursor: pointer;
			}
		}

		&-list {
			max-height: 150px;
			overflow-y: auto;
			padding: 0 8px 8px;

			&::-webkit-scrollbar {
				width: 2px;
			}
			&::-webkit-scrollbar-thumb {
				background: #666;
				border-radius: 2px;
			}
		}

		&-item {
			display: flex;
			align-items: center;
			gap: 8px;
			padding: 4px 8px;
			border-radius: 4px;
			cursor: pointer;
			font-size: 13px;
			color: #ccc;
			transition: background 0.15s;

			&:hover {
				background: rgba(255, 255, 255, 0.08);
			}

			&.select-all {
				border-bottom: 1px solid #333;
				margin-bottom: 4px;
				padding-bottom: 8px;
				font-weight: 600;
				color: #5b9bd5;
			}

			input[type="checkbox"] {
				accent-color: #5b9bd5;
				width: 14px;
				height: 14px;
				cursor: pointer;
			}

			span {
				white-space: nowrap;
				overflow: hidden;
				text-overflow: ellipsis;
			}
		}
	}

	.chat-area {
		flex: 1;
		margin: 0.25rem;
		padding: 0.75rem;
		overflow-y: auto;
		background: $bg-dark;

		.chat-message {
			padding: 4px 10px;
			margin: 0px 8px;
			border-radius: 8px;
			background-color: $bg-dark;
		}

		// 置頂訊息
		.sticky-message {
			border: 1px solid #ffffff;
			position: sticky;
			top: 0;
			z-index: 10;

			.sticky-header {
				display: flex;
				font-weight: bold;
				justify-content: space-between;
				align-items: center;
				cursor: pointer;
				padding: 8px 12px;
			}

			.sticky-body {
				padding: 8px 12px;
				font-weight: 400;
				font-size: 14px;
			}

			.toggle-btn {
				background: none;
				border: none;
				font-size: 14px;
				cursor: pointer;
				color: #ffffff;
			}
		}

		.message {
			padding: 8px;

			.bot,
			.user {
				display: flex;
				gap: 0.5rem;
				align-items: flex-start;

				&.user {
					flex-direction: row-reverse;
				}

				.avatar {
					width: 40px;
					height: 40px;
					display: flex;
					align-items: center;
					justify-content: center;
					flex-shrink: 0;

					svg {
						width: 100%;
						height: auto;
					}
				}

				.content {
					display: flex;
					flex-direction: column;
					gap: 0.5rem;

					.relation-area {
						width: 100%;
						display: flex;
						align-items: center;
						margin-top: 8px;
						margin-bottom: 8px;

						.relation-table {
							min-width: max-content;
							font-size: 13px;
						}

						.relation-table th,
						.relation-table td {
							border: 1px solid #ccc;
							text-align: left;
							padding: 0px 8px;
							line-height: 1.1;
							vertical-align: middle;
						}

						.relation-table td {
							height: 2.5rem;
						}

						.relation-table th {
							font-weight: bold;
							text-align: center;
						}
					}

					.message--bubble {
						border: 1px solid $white;
						border-radius: $radius-10;
						background: $card-bg;

						&.is-loading {
							opacity: 0.6;
							animation: pulse 1.5s ease-in-out infinite;
						}

						&.is-generative {
							border-color: #5b9bd5;
						}

						p {
							color: $white;
							white-space: pre-line;
							margin: 0;
							padding-top: 8px;
							padding-bottom: 8px;
							padding-left: 16px;
							padding-right: 16px;
							font-size: 16px;
						}
					}

					.message--button {
						display: flex;
						gap: 0.5rem;
						overflow-x: auto;

						button {
							flex-shrink: 0;
							background: $panel-bg;
							color: $white;
							font-size: 14px;
							padding: 0.5rem 1rem;
							border-radius: $radius-15;
							border: none;
							cursor: pointer;
							white-space: nowrap;

							&:hover {
								filter: brightness(0.5);
							}
						}
					}
				}
			}
		}
	}

	.input-area {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 1.5rem 1.125rem;
		background: $panel-bg;

		input[type="text"] {
			background: $white;
			height: 35px;
			width: 100%;
			border-radius: 20px;
			padding: 0 1rem;
			border: none;
			outline: none;
			color: black;

			&:disabled {
				opacity: 0.6;
				cursor: not-allowed;
			}
		}

		button {
			height: 35px;
			display: flex;
			align-items: center;
			justify-content: center;
			background: transparent;
			border: none;
			cursor: pointer;

			&:hover {
				filter: brightness(0.5);
			}

			&:disabled {
				opacity: 0.5;
				cursor: not-allowed;
			}
		}
	}
}

@keyframes pulse {
	0%, 100% { opacity: 0.6; }
	50% { opacity: 0.3; }
}
</style>
