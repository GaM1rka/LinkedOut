package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func startKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Начать интервью", "start_interview"),
		),
	)
}

func roleKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Backend", "role|Backend"),
			tgbotapi.NewInlineKeyboardButtonData("Frontend", "role|Frontend"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("QA", "role|QA"),
			tgbotapi.NewInlineKeyboardButtonData("Analyst", "role|Analyst"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Data/ML", "role|Data/ML"),
			tgbotapi.NewInlineKeyboardButtonData("Другое", "role|Другое"),
		),
	)
}

func projectTypeKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Pet-project", "ptype|Pet-project"),
			tgbotapi.NewInlineKeyboardButtonData("Учебный проект", "ptype|Учебный проект"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Хакатон", "ptype|Хакатон"),
			tgbotapi.NewInlineKeyboardButtonData("Стартап", "ptype|Стартап"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Рабочий/стажировка", "ptype|Рабочий/стажировка"),
			tgbotapi.NewInlineKeyboardButtonData("Другое", "ptype|Другое"),
		),
	)
}

func readyKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да", "ready|yes"),
			tgbotapi.NewInlineKeyboardButtonData("Скорее да", "ready|mostly"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Нет, нужны правки", "ready|no"),
		),
	)
}

func issueKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Всё ок", "issue|none"),
			tgbotapi.NewInlineKeyboardButtonData("Есть вода", "issue|water"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Есть неточность", "issue|inaccuracy"),
			tgbotapi.NewInlineKeyboardButtonData("Звучит как накрутка", "issue|exaggeration"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Другое", "issue|other"),
		),
	)
}

func paymentKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да, за один проект", "pay|one_project"),
			tgbotapi.NewInlineKeyboardButtonData("Да, за резюме целиком", "pay|full_resume"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Возможно", "pay|maybe"),
			tgbotapi.NewInlineKeyboardButtonData("Нет", "pay|none"),
		),
	)
}
