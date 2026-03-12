import json
import os
from typing import List, Optional
from models import Profile, Case, Comment

_profiles: List[Profile] = []
_purchases = {}


def load_data(file_path: str) -> None:
    """Загрузка данных из JSON"""
    global _profiles

    if not os.path.exists(file_path):
        create_test_data(file_path)

    with open(file_path, 'r', encoding='utf-8') as f:
        data = json.load(f)

    _profiles = []
    for profile_data in data.get('profile', []):
        profile = Profile.from_dict(profile_data)
        _profiles.append(profile)

    print(f"✅ Загружено {len(_profiles)} профилей")


def create_test_data(file_path: str) -> None:
    """Создание тестовых данных"""
    os.makedirs(os.path.dirname(file_path), exist_ok=True)

    test_data = {
        "profile": [
            {
                "Name": "",
                "IsCompany": False,
                "Rating": 5,
                "TgUs": "@ivan_designer",
                "Recvizits": 1234567890,
                "Cases": [
                    {
                        "Title": "Премиум дизайн сайта",
                        "Description": "Курсовая с нуля",
                        "Price": 10000,
                        "DateCreateCase": "2024-01-15"
                    },
                    {
                        "Title": "Логотип и брендинг",
                        "Description": "Разработка уникального логотипа и фирменного стиля",
                        "Price": 8000,
                        "DateCreateCase": "2024-01-20"
                    }
                ],
                "Comments": [
                    {
                        "Avtor": "@client1",
                        "Stars": 5,
                        "Title": "Отличная работа, всё сделано в срок!",
                        "DateCreateComments": "2024-02-10"
                    }
                ],
                "DateCreateProfile": "2024-01-01"
            },
            {
                "Name": "SEO-студия 'ТОП'",
                "IsCompany": True,
                "Rating": 4,
                "TgUs": "@seo_studio",
                "Recvizits": 9876543210,
                "Cases": [
                    {
                        "Title": "SEO продвижение сайта",
                        "Description": "Комплексное SEO продвижение в ТОП-10 Яндекса и Google. Аудит, оптимизация, ссылки.",
                        "Price": 25000,
                        "DateCreateCase": "2024-02-01"
                    }
                ],
                "Comments": [],
                "DateCreateProfile": "2024-01-10"
            },
            {
                "Name": "Анна Таргетолог",
                "IsCompany": False,
                "Rating": 5,
                "TgUs": "@anna_target",
                "Recvizits": 5555555555,
                "Cases": [
                    {
                        "Title": "Настройка таргетинга",
                        "Description": "Профессиональная настройка рекламы ВКонтакте и Instagram. Анализ аудитории, создание креативов.",
                        "Price": 12000,
                        "DateCreateCase": "2024-02-05"
                    },
                    {
                        "Title": "Ведение рекламного кабинета",
                        "Description": "Еженедельная оптимизация и ведение рекламных кампаний",
                        "Price": 7000,
                        "DateCreateCase": "2024-02-10"
                    }
                ],
                "Comments": [
                    {
                        "Avtor": "@kamit_pyos",
                        "Stars": 5,
                        "Title": "Лучший чертила Станкина",
                        "DateCreateComments": "2024-02-20"
                    }
                ],
                "DateCreateProfile": "2024-02-01"
            }
        ]
    }

    with open(file_path, 'w', encoding='utf-8') as f:
        json.dump(test_data, f, ensure_ascii=False, indent=2)

    print(f"✅ Создан тестовый файл: {file_path}")


def get_all_cases() -> List[dict]:
    """Получить все товары"""
    cases = []
    for profile in _profiles:
        for case in profile.cases:
            cases.append({
                'title': case.title,
                'description': case.description,
                'price': case.price,
                'date_create': case.date_create,
                'seller_name': profile.name,
                'seller_tg': profile.tg_username,
                'seller_rating': profile.rating,
                'case_obj': case,
                'profile': profile
            })
    return cases


def get_all_sellers() -> List[Profile]:
    """Получить всех продавцов"""
    return _profiles.copy()


def find_case_by_title(title: str) -> Optional[dict]:
    """Найти товар по названию"""
    for case_data in get_all_cases():
        if case_data['title'] == title:
            return case_data
    return None


def find_profile_by_tg(tg_username: str) -> Optional[Profile]:
    """Найти профиль по Telegram username"""
    for profile in _profiles:
        if profile.tg_username == tg_username:
            return profile
    return None


def save_purchase(purchase_id: str, purchase_data: dict) -> None:
    """Сохранить покупку"""
    _purchases[purchase_id] = purchase_data


def get_purchase(purchase_id: str) -> Optional[dict]:
    """Получить покупку"""
    return _purchases.get(purchase_id)