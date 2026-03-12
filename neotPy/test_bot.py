try:
    from config import BOT_TOKEN
    print(f"✅ Токен загружен: {BOT_TOKEN[:10]}...")
except Exception as e:
    print(f"❌ Ошибка импорта токена: {e}")
    BOT_TOKEN = None

if not BOT_TOKEN or BOT_TOKEN == "ВАШ_ТОКЕН_БОТА_СЮДА":
    print("❌ Токен не указан или не заменен!")
    exit()

# Простейшая команда
async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Бот работает!")

def main():
    print("🟢 Запуск тестового бота...")
    app = Application.builder().token(BOT_TOKEN).build()
    app.add_handler(CommandHandler("start", start))
    app.run_polling()

if __name__ == "__main__":
    main()