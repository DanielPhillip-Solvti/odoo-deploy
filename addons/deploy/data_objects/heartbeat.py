from pydantic import BaseModel


class EnvironmentPayload(BaseModel):
    branch: str
    status: str
    odoo_version: str = ""


class HeartbeatPayload(BaseModel):
    last_event_id: int | None
    repo_url: str
    production_branch: EnvironmentPayload
    staging_branches: list[EnvironmentPayload]
    backups: list[str] = []
    ws_url: str = ""


class EventCallbackPayload(BaseModel):
    event_id: int
    status: str
    message: str
