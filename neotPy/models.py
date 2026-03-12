from typing import List, Dict, Any, Optional
from dataclasses import dataclass
from datetime import datetime


@dataclass
class Case:
    title: str
    description: str
    price: int
    date_create: str

    def to_dict(self) -> Dict[str, Any]:
        return {
            "Title": self.title,
            "Description": self.description,
            "Price": self.price,
            "DateCreateCase": self.date_create
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'Case':
        return cls(
            title=data.get('Title', ''),
            description=data.get('Description', ''),
            price=data.get('Price', 0),
            date_create=data.get('DateCreateCase', '')
        )


@dataclass
class Comment:
    title: str
    stars: int
    author: str
    date_create: str

    def to_dict(self) -> Dict[str, Any]:
        return {
            "Title": self.title,
            "Stars": self.stars,
            "Avtor": self.author,
            "DateCreateComments": self.date_create
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'Comment':
        return cls(
            title=data.get('Title', ''),
            stars=data.get('Stars', 0),
            author=data.get('Avtor', ''),
            date_create=data.get('DateCreateComments', '')
        )


@dataclass
class Profile:
    name: str
    email: str
    is_company: bool
    rating: int
    tg_username: str
    recvizits: int
    cases: List[Case]
    comments: List[Comment]
    date_create: str

    def to_dict(self) -> Dict[str, Any]:
        return {
            "Name": self.name,
            "IsCompany": self.is_company,
            "Rating": self.rating,
            "TgUs": self.tg_username,
            "Recvizits": self.recvizits,
            "Cases": [case.to_dict() for case in self.cases],
            "Comments": [comment.to_dict() for comment in self.comments],
            "DateCreateProfile": self.date_create
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'Profile':
        cases = [Case.from_dict(case) for case in data.get('Cases', [])]
        comments = [Comment.from_dict(com) for com in data.get('Comments', [])]

        return cls(
            name=data.get('Name', ''),
            is_company=data.get('IsCompany', False),
            rating=data.get('Rating', 0),
            tg_username=data.get('TgUs', ''),
            recvizits=data.get('Recvizits', 0),
            cases=cases,
            comments=comments,
            date_create=data.get('DateCreateProfile', '')
        )


@dataclass
class Purchase:
    purchase_id: str
    buyer_id: int
    buyer_name: str
    buyer_username: Optional[str]
    case: Case
    seller_tg: str
    seller_name: str
    status: str
    created_at: str

    @classmethod
    def create(cls, purchase_id: str, buyer, case: Case, seller_tg: str, seller_name: str):
        return cls(
            purchase_id=purchase_id,
            buyer_id=buyer.id,
            buyer_name=buyer.first_name,
            buyer_username=buyer.username,
            case=case,
            seller_tg=seller_tg,
            seller_name=seller_name,
            status='pending',
            created_at=datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        )