from telegram import InlineKeyboardButton, InlineKeyboardMarkup
from typing import List, Dict, Any


def main_menu_keyboard() -> InlineKeyboardMarkup:
    """Главное меню"""
    keyboard = [
        [InlineKeyboardButton("📦 Список товаров", callback_data="menu_cases")],
        [InlineKeyboardButton("👥 Продавцы", callback_data="menu_sellers")],
        [InlineKeyboardButton("💳 Демо-оплата", callback_data="menu_payment")],
    ]
    return InlineKeyboardMarkup(keyboard)


def cases_keyboard(cases: List[Dict[str, Any]]) -> InlineKeyboardMarkup:
    """Клавиатура со списком товаров"""
    keyboard = []
    for case in cases:
        title = case['title'][:25] + '...' if len(case['title']) > 25 else case['title']
        btn_text = f"{title} - {case['price']}₽"
        keyboard.append([InlineKeyboardButton(btn_text, callback_data=f"case_{case['title']}")])

    keyboard.append([InlineKeyboardButton("🏠 Главное меню", callback_data="back_to_main")])
    return InlineKeyboardMarkup(keyboard)


def sellers_keyboard(sellers: List) -> InlineKeyboardMarkup:
    """Клавиатура со списком продавцов"""
    keyboard = []
    for seller in sellers:
        btn_text = f"{seller.name} ⭐{seller.rating}"
        keyboard.append([InlineKeyboardButton(btn_text, callback_data=f"seller_{seller.tg_username}")])

    keyboard.append([InlineKeyboardButton("🏠 Главное меню", callback_data="back_to_main")])
    return InlineKeyboardMarkup(keyboard)


def seller_detail_keyboard(seller_tg: str) -> InlineKeyboardMarkup:
    """Клавиатура для деталей продавца"""
    keyboard = [
        [InlineKeyboardButton("📦 Товары продавца", callback_data=f"seller_cases_{seller_tg}")],
        [InlineKeyboardButton("◀️ Назад к продавцам", callback_data="back_to_sellers")],
        [InlineKeyboardButton("🏠 Главное меню", callback_data="back_to_main")]
    ]
    return InlineKeyboardMarkup(keyboard)


def case_detail_keyboard(case_title: str) -> InlineKeyboardMarkup:
    """Клавиатура для деталей товара"""
    keyboard = [
        [InlineKeyboardButton("✅ Купить", callback_data=f"buy_{case_title}")],
        [InlineKeyboardButton("◀️ Назад к товарам", callback_data="back_to_cases")],
        [InlineKeyboardButton("🏠 Главное меню", callback_data="back_to_main")]
    ]
    return InlineKeyboardMarkup(keyboard)


def payment_keyboard() -> InlineKeyboardMarkup:
    """Клавиатура для демо-оплаты"""
    keyboard = [
        [InlineKeyboardButton("💰 Шпилька-1500", callback_data="pay_1500")],
        [InlineKeyboardButton("💰 Вал - 2000 ", callback_data="pay_2000")],
        [InlineKeyboardButton("💰 Курсовая - 10000", callback_data="pay_10000")],
        [InlineKeyboardButton("🏠 Главное меню", callback_data="back_to_main")]
    ]
    return InlineKeyboardMarkup(keyboard)


def payment_demo_keyboard(purchase_id: str) -> InlineKeyboardMarkup:
    """Клавиатура для имитации оплаты"""
    keyboard = [
        [InlineKeyboardButton("✅ Имитировать успешную оплату", callback_data=f"success_{purchase_id}")],
        [InlineKeyboardButton("❌ Отмена", callback_data="back_to_payment")],
        [InlineKeyboardButton("🏠 Главное меню", callback_data="back_to_main")]
    ]
    return InlineKeyboardMarkup(keyboard)


def back_to_main_keyboard() -> InlineKeyboardMarkup:
    """Клавиатура только с кнопкой возврата"""
    keyboard = [[InlineKeyboardButton("🏠 Главное меню", callback_data="back_to_main")]]
    return InlineKeyboardMarkup(keyboard)