import logging
import os
from telegram import Update, BotCommand
from telegram.ext import Application, CommandHandler, CallbackQueryHandler, ContextTypes

from config import BOT_TOKEN, DATA_FILE
from database import load_data
from handlers import start_command, sellers_command, cases_command, payment_command, button_handler

# Настройка логирования
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def set_commands(app: Application):
    """Установка команд бота"""
    commands = [
        BotCommand("start", "🏠 Главное меню"),
        BotCommand("cases", "📦 Список товаров"),
        BotCommand("sellers", "👥 Продавцы"),
        BotCommand("payment", "💳 Оплата")
    ]
    await app.bot.set_my_commands(commands)
    logger.info("✅ Команды установлены")


def main():
    print("\n" + "=" * 50)
    print("🤖 БОТ Неотчислят")
    print("=" * 50)

    # Проверка токена
    if not BOT_TOKEN or BOT_TOKEN == "8262205111:AAEIqiIQ0_rLvOOG1Rd7EenWWochlh1GtnY" and BOT_TOKEN.count(':') != 1:
        print("\n❌ ОШИБКА: Неправильный токен в config.py!")
        return

    print(f"✅ Токен загружен")

    # Загрузка данных
    if os.path.exists(DATA_FILE):
        load_data(DATA_FILE)
        print(f"✅ Данные загружены из {DATA_FILE}")
    else:
        print(f"⚠️ Файл {DATA_FILE} не найден, будут использованы тестовые данные")
        from database import create_test_data
        create_data = getattr(create_test_data, '__wrapped__', create_test_data)
        create_data(DATA_FILE)
        load_data(DATA_FILE)

    # Создание приложения
    app = Application.builder().token(BOT_TOKEN).post_init(set_commands).build()

    # Регистрация обработчиков команд
    app.add_handler(CommandHandler('start', start_command))
    app.add_handler(CommandHandler('cases', cases_command))
    app.add_handler(CommandHandler('sellers', sellers_command))
    app.add_handler(CommandHandler('payment', payment_command))

    # Регистрация обработчика кнопок
    app.add_handler(CallbackQueryHandler(button_handler))

    print("\n📋 Доступные команды:")
    print("   /start    - 🏠 Главное меню")
    print("   /cases    - 📦 Список товаров")
    print("   /sellers  - 👥 Продавцы")
    print("   /payment  - 💳 Оплата")
    print("\n🟢 Бот запущен. Нажмите Ctrl+C для остановки.\n")

    app.run_polling()


if __name__ == '__main__':
    main()