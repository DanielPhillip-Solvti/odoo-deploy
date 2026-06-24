from pydantic import BaseModel

class EnvironmentPayload(BaseModel):
    branch: str
    status: str

class HeartbeatPayload(BaseModel):
    last_event_id: int | None
    repo_url: str
    production_branch: EnvironmentPayload
    staging_branches: list[EnvironmentPayload]

class EventCallbackPayload(BaseModel):
    status: str
    message: str