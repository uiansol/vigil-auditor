from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AuditRequest(_message.Message):
    __slots__ = ("audit_id", "file_name", "content_type", "redacted_content")
    AUDIT_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    REDACTED_CONTENT_FIELD_NUMBER: _ClassVar[int]
    audit_id: str
    file_name: str
    content_type: str
    redacted_content: bytes
    def __init__(self, audit_id: _Optional[str] = ..., file_name: _Optional[str] = ..., content_type: _Optional[str] = ..., redacted_content: _Optional[bytes] = ...) -> None: ...

class AuditEvent(_message.Message):
    __slots__ = ("stage", "subscription_found", "summary", "result", "error")
    STAGE_FIELD_NUMBER: _ClassVar[int]
    SUBSCRIPTION_FOUND_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    stage: StageEvent
    subscription_found: SubscriptionFound
    summary: SummaryEvent
    result: AuditResult
    error: ErrorEvent
    def __init__(self, stage: _Optional[_Union[StageEvent, _Mapping]] = ..., subscription_found: _Optional[_Union[SubscriptionFound, _Mapping]] = ..., summary: _Optional[_Union[SummaryEvent, _Mapping]] = ..., result: _Optional[_Union[AuditResult, _Mapping]] = ..., error: _Optional[_Union[ErrorEvent, _Mapping]] = ...) -> None: ...

class StageEvent(_message.Message):
    __slots__ = ("stage", "message")
    STAGE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    stage: str
    message: str
    def __init__(self, stage: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class SubscriptionFound(_message.Message):
    __slots__ = ("raw_merchant_name", "normalized_name", "billing_frequency", "current_amount", "previous_amount", "is_price_creep", "confidence_score", "first_detected_date", "last_detected_date", "friction_level")
    RAW_MERCHANT_NAME_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_NAME_FIELD_NUMBER: _ClassVar[int]
    BILLING_FREQUENCY_FIELD_NUMBER: _ClassVar[int]
    CURRENT_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    IS_PRICE_CREEP_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_SCORE_FIELD_NUMBER: _ClassVar[int]
    FIRST_DETECTED_DATE_FIELD_NUMBER: _ClassVar[int]
    LAST_DETECTED_DATE_FIELD_NUMBER: _ClassVar[int]
    FRICTION_LEVEL_FIELD_NUMBER: _ClassVar[int]
    raw_merchant_name: str
    normalized_name: str
    billing_frequency: str
    current_amount: float
    previous_amount: float
    is_price_creep: bool
    confidence_score: float
    first_detected_date: str
    last_detected_date: str
    friction_level: str
    def __init__(self, raw_merchant_name: _Optional[str] = ..., normalized_name: _Optional[str] = ..., billing_frequency: _Optional[str] = ..., current_amount: _Optional[float] = ..., previous_amount: _Optional[float] = ..., is_price_creep: bool = ..., confidence_score: _Optional[float] = ..., first_detected_date: _Optional[str] = ..., last_detected_date: _Optional[str] = ..., friction_level: _Optional[str] = ...) -> None: ...

class SummaryEvent(_message.Message):
    __slots__ = ("total_monthly_spend", "projected_annual_cost", "price_spike_count")
    TOTAL_MONTHLY_SPEND_FIELD_NUMBER: _ClassVar[int]
    PROJECTED_ANNUAL_COST_FIELD_NUMBER: _ClassVar[int]
    PRICE_SPIKE_COUNT_FIELD_NUMBER: _ClassVar[int]
    total_monthly_spend: float
    projected_annual_cost: float
    price_spike_count: int
    def __init__(self, total_monthly_spend: _Optional[float] = ..., projected_annual_cost: _Optional[float] = ..., price_spike_count: _Optional[int] = ...) -> None: ...

class AuditResult(_message.Message):
    __slots__ = ("audit_id", "status", "failure_reason", "summary", "subscriptions")
    AUDIT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SUBSCRIPTIONS_FIELD_NUMBER: _ClassVar[int]
    audit_id: str
    status: str
    failure_reason: str
    summary: SummaryEvent
    subscriptions: _containers.RepeatedCompositeFieldContainer[SubscriptionFound]
    def __init__(self, audit_id: _Optional[str] = ..., status: _Optional[str] = ..., failure_reason: _Optional[str] = ..., summary: _Optional[_Union[SummaryEvent, _Mapping]] = ..., subscriptions: _Optional[_Iterable[_Union[SubscriptionFound, _Mapping]]] = ...) -> None: ...

class ErrorEvent(_message.Message):
    __slots__ = ("code", "message")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...
