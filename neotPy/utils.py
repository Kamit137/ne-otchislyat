# utils.py
def format_price(price: int) -> str:
    """Форматирование цены"""
    return f"{price:,} ₽".replace(",", " ")

def truncate_text(text: str, max_length: int = 100) -> str:
    """Обрезать текст"""
    if len(text) <= max_length:
        return text
    return text[:max_length-3] + "..."