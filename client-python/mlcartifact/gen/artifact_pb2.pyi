from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WriteRequest(_message.Message):
    __slots__ = ("filename", "content", "mime_type", "expires_hours", "source", "metadata", "user_id", "description", "virtual_path")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_HOURS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VIRTUAL_PATH_FIELD_NUMBER: _ClassVar[int]
    filename: str
    content: bytes
    mime_type: str
    expires_hours: int
    source: str
    metadata: _containers.ScalarMap[str, str]
    user_id: str
    description: str
    virtual_path: str
    def __init__(self, filename: _Optional[str] = ..., content: _Optional[bytes] = ..., mime_type: _Optional[str] = ..., expires_hours: _Optional[int] = ..., source: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ..., user_id: _Optional[str] = ..., description: _Optional[str] = ..., virtual_path: _Optional[str] = ...) -> None: ...

class WriteResponse(_message.Message):
    __slots__ = ("id", "filename", "uri", "expires_at", "virtual_path")
    ID_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    URI_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    VIRTUAL_PATH_FIELD_NUMBER: _ClassVar[int]
    id: str
    filename: str
    uri: str
    expires_at: str
    virtual_path: str
    def __init__(self, id: _Optional[str] = ..., filename: _Optional[str] = ..., uri: _Optional[str] = ..., expires_at: _Optional[str] = ..., virtual_path: _Optional[str] = ...) -> None: ...

class ReadRequest(_message.Message):
    __slots__ = ("id", "user_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    user_id: str
    def __init__(self, id: _Optional[str] = ..., user_id: _Optional[str] = ...) -> None: ...

class ReadResponse(_message.Message):
    __slots__ = ("content", "mime_type", "filename", "virtual_path")
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    VIRTUAL_PATH_FIELD_NUMBER: _ClassVar[int]
    content: bytes
    mime_type: str
    filename: str
    virtual_path: str
    def __init__(self, content: _Optional[bytes] = ..., mime_type: _Optional[str] = ..., filename: _Optional[str] = ..., virtual_path: _Optional[str] = ...) -> None: ...

class DeleteRequest(_message.Message):
    __slots__ = ("id", "user_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    user_id: str
    def __init__(self, id: _Optional[str] = ..., user_id: _Optional[str] = ...) -> None: ...

class DeleteResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class ListRequest(_message.Message):
    __slots__ = ("source", "user_id", "limit", "offset", "dir_path")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    DIR_PATH_FIELD_NUMBER: _ClassVar[int]
    source: str
    user_id: str
    limit: int
    offset: int
    dir_path: str
    def __init__(self, source: _Optional[str] = ..., user_id: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ..., dir_path: _Optional[str] = ...) -> None: ...

class ListResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[ArtifactInfo]
    def __init__(self, items: _Optional[_Iterable[_Union[ArtifactInfo, _Mapping]]] = ...) -> None: ...

class ArtifactInfo(_message.Message):
    __slots__ = ("id", "filename", "mime_type", "source", "created_at", "expires_at", "size_bytes", "user_id", "description", "virtual_path", "is_directory")
    ID_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VIRTUAL_PATH_FIELD_NUMBER: _ClassVar[int]
    IS_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    id: str
    filename: str
    mime_type: str
    source: str
    created_at: str
    expires_at: str
    size_bytes: int
    user_id: str
    description: str
    virtual_path: str
    is_directory: bool
    def __init__(self, id: _Optional[str] = ..., filename: _Optional[str] = ..., mime_type: _Optional[str] = ..., source: _Optional[str] = ..., created_at: _Optional[str] = ..., expires_at: _Optional[str] = ..., size_bytes: _Optional[int] = ..., user_id: _Optional[str] = ..., description: _Optional[str] = ..., virtual_path: _Optional[str] = ..., is_directory: _Optional[bool] = ...) -> None: ...

class PatchRequest(_message.Message):
    __slots__ = ("id", "user_id", "content", "line_start", "line_end", "append")
    ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    LINE_START_FIELD_NUMBER: _ClassVar[int]
    LINE_END_FIELD_NUMBER: _ClassVar[int]
    APPEND_FIELD_NUMBER: _ClassVar[int]
    id: str
    user_id: str
    content: bytes
    line_start: int
    line_end: int
    append: bool
    def __init__(self, id: _Optional[str] = ..., user_id: _Optional[str] = ..., content: _Optional[bytes] = ..., line_start: _Optional[int] = ..., line_end: _Optional[int] = ..., append: _Optional[bool] = ...) -> None: ...

class PatchResponse(_message.Message):
    __slots__ = ("success", "new_size", "updated_at")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    NEW_SIZE_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    new_size: int
    updated_at: str
    def __init__(self, success: _Optional[bool] = ..., new_size: _Optional[int] = ..., updated_at: _Optional[str] = ...) -> None: ...

class FindRequest(_message.Message):
    __slots__ = ("user_id", "pattern")
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    user_id: str
    pattern: str
    def __init__(self, user_id: _Optional[str] = ..., pattern: _Optional[str] = ...) -> None: ...
