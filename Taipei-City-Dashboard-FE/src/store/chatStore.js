import { ref, watch } from 'vue'
import { defineStore } from 'pinia'
import http from "../router/axios";

export const useChatStore = defineStore('chat', () => {
	// 模式切換：'search' = 組件搜尋, 'generative' = 生成式問答
	const chatMode = ref('search');

	// 生成式模式：勾選的組件 ids
	const selectedComponentIds = ref([]);

	// 生成式模式：是否正在等待回應
	const isGenerating = ref(false);

	// 預設訊息
	const defaultChatData = [
	{
		id: 1,
		role: 'bot',
		isDefault: true,
		content:
		'您好，我是【臺北城市儀表板】小幫手，很高興為您服務！\n 您可以： \n\n • 點擊左側既有的儀表板主題，快速查看各主題內容 \n • 輸入您感興趣的主題描述，我會自動為您組建最適合的儀表板 \n • 開啟「數據問答」模式，勾選組件後直接提問 \n\n 如果有想了解的內容，歡迎直接告訴我，我會盡力協助！\n\n 📩 聯絡信箱：tuic@gov.taipei \n 🏢 臺北大數據中心 \n\n',
	},
	];

	const recommendComponents = ref(null)

	// 從 sessionStorage 讀取
	const savedChatData = JSON.parse(sessionStorage.getItem('chatData')) || [];

	// 拼接預設訊息 + sessionStorage 的聊天紀錄
	const chatData = ref([...defaultChatData, ...savedChatData]);

	// 監聽 chatData 的變化，自動同步到 sessionStorage
	watch(
	chatData,
	(newVal) => {
	const userBotMessages = newVal.filter((item) => !item.isDefault)
	sessionStorage.setItem('chatData', JSON.stringify(userBotMessages))
	},
	{ deep: true }
	);

	const addChatData = (newChatData) => {
	chatData.value.push({ id: chatData.value.length + 1, isDefault: false, ...newChatData });
	};

	// 原有的搜尋模式
	const addQueryData = async (newChatData) => {

	chatData.value.push({ id: chatData.value.length + 1, isDefault: false, ...newChatData });

		recommendComponents.value = [];
		let topK = null;

		try {
			const response = await http.post(
				"/vector/component",
				new URLSearchParams({
				query: newChatData.content,
				limit: 10,
				score: 0.8,
				}),
				{
				headers: {
					"Content-Type": "application/x-www-form-urlencoded",
				},
				}
			);
			if (response.data?.data?.length > 0) {
				recommendComponents.value = response.data.data;
			}

			const result = Array.from(
				recommendComponents.value.reduce((map, item) => {
				const key = item.index
				const exist = map.get(key)

				if (!exist) {
					map.set(key, item)
					return map
				}

				if (item.city === 'metrotaipei') {
					map.set(key, item)
				}

				return map
				}, new Map()).values()
			)
			recommendComponents.value = result

		} catch (error) {
			console.error("VectorAnalysisError :", error);
		}

		if (recommendComponents.value && recommendComponents.value?.length > 0) {
			topK = [...recommendComponents.value].sort((a, b) => b.score - a.score);
			chatData.value.push({ id: chatData.value.length + 1, role: 'bot', isDefault: false, button: [{ id:1, text:'建立儀表板' }], content: `您好 😊 \n 以下是根據您的問題，自動為您推薦的「組件清單」。您可以將這些組件整批加入「個人儀表板」，方便日後快速查看與使用。\n`, relations: topK });
			chatData.value.push({ id: chatData.value.length + 1, role: 'bot', isDefault: false, content: `若您有任何新的查詢或想深入探索的內容，都可以隨時在對話框告訴我～\n 我很樂意再協助您 💬✨` });
		} else {
			chatData.value.push({ id: chatData.value.length + 1, role: 'bot', isDefault: false, content: `很抱歉，您提供的描述沒有相似組件，請繼續提問 ! ` });
		}

		saveChatLog(newChatData.content, recommendComponents.value);
	};

	// 新增：生成式問答模式
	const addGenerativeQuery = async (newChatData, components) => {
		// 加入使用者訊息
		chatData.value.push({ id: chatData.value.length + 1, isDefault: false, ...newChatData });

		// 顯示載入中
		isGenerating.value = true;
		const loadingId = chatData.value.length + 1;
		chatData.value.push({ id: loadingId, role: 'bot', isDefault: false, content: '思考中...', isLoading: true });

		try {
			// 1. 收集勾選組件的數據
			const componentDataList = [];
			for (const comp of components) {
				if (selectedComponentIds.value.includes(comp.id)) {
					const dataStr = formatComponentData(comp);
					if (dataStr) {
						componentDataList.push(dataStr);
					}
				}
			}

			if (componentDataList.length === 0) {
				// 移除載入訊息
				chatData.value = chatData.value.filter(c => c.id !== loadingId);
				chatData.value.push({ id: chatData.value.length + 1, role: 'bot', isDefault: false, content: '請先勾選至少一個組件作為參考資料。' });
				isGenerating.value = false;
				return;
			}

			// 2. 組成 prompt
			const systemPrompt = `你是臺北城市儀表板的數據分析助手。使用者會根據儀表板上的組件數據提問，請根據提供的數據回答問題。
回答要求：
- 使用繁體中文
- 回答要簡潔明瞭
- 如果數據中有具體數字，請引用
- 如果數據不足以回答問題，請誠實說明`;

			const dataContext = `以下是使用者勾選的儀表板組件數據：\n\n${componentDataList.join('\n\n')}`;

			// 3. 呼叫 TWCC AI API
			const response = await http.post("/ai/chat/twai", {
				messages: [
					{ role: "system", content: systemPrompt },
					{ role: "user", content: `${dataContext}\n\n使用者提問：${newChatData.content}` }
				]
			});

			// 移除載入訊息
			chatData.value = chatData.value.filter(c => c.id !== loadingId);

			if (response.data?.data?.content) {
				chatData.value.push({
					id: chatData.value.length + 1,
					role: 'bot',
					isDefault: false,
					content: response.data.data.content,
					isGenerative: true,
				});
			} else {
				chatData.value.push({
					id: chatData.value.length + 1,
					role: 'bot',
					isDefault: false,
					content: '抱歉，AI 服務暫時無法回應，請稍後再試。',
				});
			}
		} catch (error) {
			console.error("GenerativeAI Error:", error);
			// 移除載入訊息
			chatData.value = chatData.value.filter(c => c.id !== loadingId);
			chatData.value.push({
				id: chatData.value.length + 1,
				role: 'bot',
				isDefault: false,
				content: '抱歉，AI 服務發生錯誤，請稍後再試。',
			});
		} finally {
			isGenerating.value = false;
		}

		saveChatLog(newChatData.content, "generative");
	};

	// 把組件的 chart_data 格式化成文字給 LLM 讀
	function formatComponentData(comp) {
		if (!comp || !comp.chart_data) return null;

		let result = `【${comp.name}】（資料來源：${comp.source || '未知'}）\n`;

		const data = comp.chart_data;

		// two_d 格式：[{data: [{x, y}, ...]}]
		if (Array.isArray(data) && data[0]?.data && data[0].data[0]?.x !== undefined) {
			result += data[0].data.map(d => `  ${d.x}: ${d.y}`).join('\n');
			if (comp.chart_config?.unit) {
				result += `\n  單位：${comp.chart_config.unit}`;
			}
			return result;
		}

		// three_d 格式：有 categories
		if (comp.chart_config?.categories && Array.isArray(data)) {
			const categories = comp.chart_config.categories;
			result += `  類別：${categories.join(', ')}\n`;
			data.forEach(series => {
				if (series.name && series.data) {
					result += `  ${series.name}：${series.data.join(', ')}\n`;
				}
			});
			if (comp.chart_config?.unit) {
				result += `  單位：${comp.chart_config.unit}`;
			}
			return result;
		}

		// time 格式：[{name, data: [{x, y}, ...]}]
		if (Array.isArray(data) && data[0]?.name && data[0]?.data?.[0]?.x) {
			data.forEach(series => {
				result += `  ${series.name}：\n`;
				series.data.forEach(d => {
					const timeStr = typeof d.x === 'string' ? d.x.slice(0, 10) : d.x;
					result += `    ${timeStr}: ${d.y}\n`;
				});
			});
			if (comp.chart_config?.unit) {
				result += `  單位：${comp.chart_config.unit}`;
			}
			return result;
		}

		// fallback：直接 JSON
		try {
			result += `  ${JSON.stringify(data).slice(0, 500)}`;
		} catch (e) {
			result += '  (資料格式無法解析)';
		}
		return result;
	}

	const saveChatLog = async(question, answer) => {
		try {
		const formData = new FormData();
		const d = new Date();
		const todayId =
			d.getFullYear() +
			String(d.getMonth() + 1).padStart(2, "0") +
			String(d.getDate()).padStart(2, "0");

		formData.append("session", "session_" + todayId);
		formData.append("question", question);
		formData.append("answer", JSON.stringify(answer));

		await http.post("/chatlog/", formData, {
			headers: {
			"Content-Type": "multipart/form-data",
			},
		});
	} catch (error) {
		console.error("saveChatLog error:", error);
	}
	};

	// 切換模式
	const toggleMode = (mode) => {
		chatMode.value = mode;
	};

	// 勾選/取消勾選組件
	const toggleComponent = (id) => {
		const idx = selectedComponentIds.value.indexOf(id);
		if (idx >= 0) {
			selectedComponentIds.value.splice(idx, 1);
		} else {
			selectedComponentIds.value.push(id);
		}
	};

	// 全選/全不選
	const selectAllComponents = (ids) => {
		selectedComponentIds.value = [...ids];
	};

	const deselectAllComponents = () => {
		selectedComponentIds.value = [];
	};

	return {
		chatData,
		chatMode,
		selectedComponentIds,
		isGenerating,
		addChatData,
		addQueryData,
		addGenerativeQuery,
		saveChatLog,
		toggleMode,
		toggleComponent,
		selectAllComponents,
		deselectAllComponents,
	}
})
