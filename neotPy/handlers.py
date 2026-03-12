# handlers.py
import uuid
from telegram import Update
from telegram.ext import ContextTypes

from database import get_all_cases, get_all_sellers, find_case_by_title, find_profile_by_tg, save_purchase
from models import Purchase
from keyboards import *
from utils import format_price, truncate_text


# ============== КОМАНДА START ==============
async def start_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """🏠 Главное меню"""
    text = (
        f"🏠 **ГЛАВНОЕ МЕНЮ**\n\n"
        f"Добро пожаловать! Это демо-бот для ЮKassa.\n\n"
        f"📌 **Доступные команды:**\n"
        f"• /cases - список товаров\n"
        f"• /sellers - продавцы\n"
        f"• /payment - демо-оплата\n\n"
        f"Выберите раздел:"
    )

    await update.message.reply_text(
        text,
        reply_markup=main_menu_keyboard(),
        parse_mode='Markdown'
    )


# ============== КОМАНДА CASES ==============
async def cases_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """📦 Список товаров"""
    cases = get_all_cases()

    if not cases:
        await update.message.reply_text("😔 Товары временно отсутствуют")
        return

    text = "📦 **СПИСОК ТОВАРОВ**\n\n"
    for i, case in enumerate(cases, 1):
        text += f"{i}. **{case['title']}**\n"
        text += f"   💰 {format_price(case['price'])}\n"
        text += f"   👤 {case['seller_name']}\n\n"

    await update.message.reply_text(
        text,
        reply_markup=cases_keyboard(cases),
        parse_mode='Markdown'
    )


# ============== КОМАНДА SELLERS ==============
async def sellers_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """👥 Продавцы"""
    sellers = get_all_sellers()

    if not sellers:
        await update.message.reply_text("😔 Продавцы временно отсутствуют")
        return

    text = "👥 **ПРОДАВЦЫ**\n\n"
    for seller in sellers:
        text += f"**{seller.name}** ⭐{seller.rating}/5\n"
        text += f"📱 {seller.tg_username}\n"
        text += f"📦 Товаров: {len(seller.cases)}\n\n"

    await update.message.reply_text(
        text,
        reply_markup=sellers_keyboard(sellers),
        parse_mode='Markdown'
    )


# ============== КОМАНДА PAYMENT ==============
async def payment_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """💳 Демо-оплата"""
    text = (
        "💳 **ДЕМОНСТРАЦИЯ ОПЛАТЫ**\n\n"
        "Это имитация оплаты для показа ЮKassa.\n\n"
        "**В реальном боте здесь будет:**\n"
        "• Окно оплаты Telegram Payments\n"
        "• Выбор способа оплаты\n"
        "• Подтверждение платежа\n\n"
        "Выберите сумму для демо:"
    )

    await update.message.reply_text(
        text,
        reply_markup=payment_keyboard(),
        parse_mode='Markdown'
    )


# ============== ОБРАБОТЧИК КНОПОК ==============
async def button_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Единый обработчик всех кнопок"""
    query = update.callback_query
    await query.answer()

    data = query.data
    print(f"📩 Callback: {data}")

    # ===== НАВИГАЦИЯ =====
    if data == "back_to_main":
        text = "🏠 **ГЛАВНОЕ МЕНЮ**\n\nВыберите раздел:"
        await query.edit_message_text(
            text,
            reply_markup=main_menu_keyboard(),
            parse_mode='Markdown'
        )
        return

    if data == "back_to_cases":
        cases = get_all_cases()
        text = "📦 **СПИСОК ТОВАРОВ**\n\n"
        for i, case in enumerate(cases, 1):
            text += f"{i}. **{case['title']}** - {format_price(case['price'])}\n"
        await query.edit_message_text(
            text,
            reply_markup=cases_keyboard(cases),
            parse_mode='Markdown'
        )
        return

    if data == "back_to_sellers":
        sellers = get_all_sellers()
        text = "👥 **ПРОДАВЦЫ**\n\n"
        for seller in sellers:
            text += f"**{seller.name}** ⭐{seller.rating}/5\n"
            text += f"📱 {seller.tg_username}\n\n"
        await query.edit_message_text(
            text,
            reply_markup=sellers_keyboard(sellers),
            parse_mode='Markdown'
        )
        return

    if data == "back_to_payment":
        text = "💳 **ДЕМОНСТРАЦИЯ ОПЛАТЫ**\n\nВыберите сумму:"
        await query.edit_message_text(
            text,
            reply_markup=payment_keyboard(),
            parse_mode='Markdown'
        )
        return

    # ===== МЕНЮ ИЗ ГЛАВНОГО ЭКРАНА =====
    if data == "menu_cases":
        cases = get_all_cases()
        text = "📦 **СПИСОК ТОВАРОВ**\n\n"
        for i, case in enumerate(cases, 1):
            text += f"{i}. **{case['title']}** - {format_price(case['price'])}\n"
        await query.edit_message_text(
            text,
            reply_markup=cases_keyboard(cases),
            parse_mode='Markdown'
        )
        return

    if data == "menu_sellers":
        sellers = get_all_sellers()
        text = "👥 **ПРОДАВЦЫ**\n\n"
        for seller in sellers:
            text += f"**{seller.name}** ⭐{seller.rating}/5\n"
            text += f"📱 {seller.tg_username}\n\n"
        await query.edit_message_text(
            text,
            reply_markup=sellers_keyboard(sellers),
            parse_mode='Markdown'
        )
        return

    if data == "menu_payment":
        text = "💳 **ДЕМОНСТРАЦИЯ ОПЛАТЫ**\n\nВыберите сумму:"
        await query.edit_message_text(
            text,
            reply_markup=payment_keyboard(),
            parse_mode='Markdown'
        )
        return

    # ===== ТОВАРЫ =====
    if data.startswith("case_"):
        title = data[5:]
        case_data = find_case_by_title(title)

        if not case_data:
            await query.edit_message_text("❌ Товар не найден")
            return

        text = (
            f"📌 **{case_data['title']}**\n\n"
            f"📝 {truncate_text(case_data['description'], 200)}\n\n"
            f"💰 **Цена:** {format_price(case_data['price'])}\n"
            f"👤 **Продавец:** {case_data['seller_name']}\n"
            f"📱 {case_data['seller_tg']}\n"
            f"⭐ Рейтинг: {case_data['seller_rating']}/5\n"
            f"📅 Дата: {case_data.get('date_create', 'Не указана')}"
        )

        await query.edit_message_text(
            text,
            reply_markup=case_detail_keyboard(case_data['title']),
            parse_mode='Markdown'
        )
        return

    # ===== ПОКУПКА =====
    if data.startswith("buy_"):
        title = data[4:]
        case_data = find_case_by_title(title)

        if not case_data:
            await query.edit_message_text("❌ Товар не найден")
            return

        purchase_id = str(uuid.uuid4())[:8]
        purchase = Purchase.create(
            purchase_id=purchase_id,
            buyer=query.from_user,
            case=case_data['case_obj'],
            seller_tg=case_data['seller_tg'],
            seller_name=case_data['seller_name']
        )

        save_purchase(purchase_id, {
            'id': purchase_id,
            'buyer_id': purchase.buyer_id,
            'buyer_name': purchase.buyer_name,
            'buyer_username': purchase.buyer_username,
            'case': {
                'title': purchase.case.title,
                'price': purchase.case.price
            },
            'seller_tg': purchase.seller_tg,
            'seller_name': purchase.seller_name
        })

        text = (
            f"🧾 **ДЕМОНСТРАЦИЯ ОПЛАТЫ**\n\n"
            f"Товар: **{case_data['title']}**\n"
            f"Сумма: **{format_price(case_data['price'])}**\n"
            f"Продавец: {case_data['seller_name']}\n\n"
            f"━━━━━━━━━━━━━━━━━━━━\n\n"
            f"💳 **В реальном боте здесь будет окно оплаты Telegram**\n\n"
            f"━━━━━━━━━━━━━━━━━━━━\n"
            f"🆔 Демо-ID: `{purchase_id}`"
        )

        await query.edit_message_text(
            text,
            reply_markup=payment_demo_keyboard(purchase_id),
            parse_mode='Markdown'
        )
        return

    # ===== УСПЕШНАЯ ОПЛАТА =====
    if data.startswith("success_"):
        purchase_id = data[8:]
        from database import get_purchase
        purchase = get_purchase(purchase_id)

        if not purchase:
            await query.edit_message_text("❌ Покупка не найдена")
            return

        case = purchase['case']

        text = (
            f"✅ **ОПЛАТА УСПЕШНО ВЫПОЛНЕНА!**\n\n"
            f"Товар: **{case['title']}**\n"
            f"Сумма: **{format_price(case['price'])}**\n"
            f"Продавец: {purchase['seller_name']}\n\n"
            f"━━━━━━━━━━━━━━━━━━━━\n\n"
            f"📱 **В реальном боте:**\n"
            f"✓ Продавец {purchase['seller_tg']} получил уведомление\n"
            f"✓ Платеж проведен через ЮKassa\n\n"
            f"━━━━━━━━━━━━━━━━━━━━\n\n"
            f"📸 **Этот экран для ЮKassa**\n"
            f"ID платежа: `{purchase_id}`"
        )

        await query.edit_message_text(
            text,
            reply_markup=back_to_main_keyboard(),
            parse_mode='Markdown'
        )

        # Уведомление продавцу
        await context.bot.send_message(
            chat_id=query.message.chat_id,
            text=(
                f"📨 **УВЕДОМЛЕНИЕ ПРОДАВЦУ**\n\n"
                f"Продавец {purchase['seller_name']} ({purchase['seller_tg']}) "
                f"получил сообщение:\n\n"
                f"\"💰 Новая продажа! Товар {case['title']} "
                f"на сумму {format_price(case['price'])}. "
                f"Покупатель: @{purchase.get('buyer_username', 'unknown')}\""
            ),
            parse_mode='Markdown'
        )
        return

    # ===== ПРОДАВЦЫ =====
    if data.startswith("seller_"):
        tg = data[7:]
        profile = find_profile_by_tg(tg)

        if not profile:
            await query.edit_message_text("❌ Продавец не найден")
            return

        text = (
            f"👤 **{profile.name}**\n\n"
            f"⭐ Рейтинг: {profile.rating}/5\n"
            f"📱 {profile.tg_username}\n"
            f"📧 {profile.email}\n"
            f"🏢 Компания: {'Да' if profile.is_company else 'Нет'}\n"
            f"📦 Товаров: {len(profile.cases)}\n"
            f"💬 Отзывов: {len(profile.comments)}\n"
            f"📅 На сайте с: {profile.date_create}\n\n"
        )

        if profile.comments:
            text += "**Последний отзыв:**\n"
            last_comment = profile.comments[-1]
            text += f"\"{last_comment.title}\" — {last_comment.author} ⭐{last_comment.stars}\n"

        await query.edit_message_text(
            text,
            reply_markup=seller_detail_keyboard(profile.tg_username),
            parse_mode='Markdown'
        )
        return

    if data.startswith("seller_cases_"):
        tg = data[13:]
        profile = find_profile_by_tg(tg)

        if not profile:
            await query.edit_message_text("❌ Продавец не найден")
            return

        if not profile.cases:
            await query.edit_message_text(f"У продавца {profile.name} пока нет товаров")
            return

        text = f"📦 **Товары продавца {profile.name}**\n\n"
        for i, case in enumerate(profile.cases, 1):
            text += f"{i}. **{case.title}** - {format_price(case.price)}\n"
            text += f"   {truncate_text(case.description, 50)}\n\n"

        # Создаем клавиатуру с товарами этого продавца
        keyboard = []
        for case in profile.cases:
            btn_text = f"{case.title[:20]} - {case.price}₽"
            keyboard.append([InlineKeyboardButton(btn_text, callback_data=f"case_{case.title}")])

        keyboard.append([InlineKeyboardButton("◀️ Назад к продавцу", callback_data=f"seller_{tg}")])
        keyboard.append([InlineKeyboardButton("🏠 Главное меню", callback_data="back_to_main")])

        await query.edit_message_text(
            text,
            reply_markup=InlineKeyboardMarkup(keyboard),
            parse_mode='Markdown'
        )
        return

    # ===== ДЕМО-ПЛАТЕЖИ =====
    if data.startswith("pay_"):
        amount = data[4:]

        text = (
            f"🧾 **ДЕМОНСТРАЦИЯ ОПЛАТЫ**\n\n"
            f"Сумма: **{amount} ₽**\n\n"
            f"━━━━━━━━━━━━━━━━━━━━\n\n"
            f"💳 **В реальном боте здесь будет окно оплаты Telegram**\n\n"
            f"━━━━━━━━━━━━━━━━━━━━\n"
            f"🆔 Демо-ID: `{uuid.uuid4().hex[:8]}`"
        )

        await query.edit_message_text(
            text,
            reply_markup=back_to_main_keyboard(),
            parse_mode='Markdown'
        )
        return

    # Если дошли сюда - неизвестный callback
    print(f"⚠️ Неизвестный callback: {data}")
    await query.edit_message_text(
        "🔄 Обновление интерфейса...",
        reply_markup=main_menu_keyboard()
    )