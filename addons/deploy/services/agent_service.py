ACTIONS = [
    ('deploy', 'Deploy'),
    ('undeploy', 'Undeploy'),
    ('backup', 'Backup'),
    ('restore_backup', 'Restore Backup'),
    ('reset_branch', 'Reset Branch'),
    ('update_module', 'Update Module'),
    ('download_dump', 'Download Dump'),
    ('stream_logs', 'Stream Logs'),
]
    
class AgentService:
    def __init__(self, agent):
        agent.ensure_one()
        self.agent = agent

    def queue_action(self, action, parameters=None):
        if action not in [a[0] for a in ACTIONS]:
            raise ValueError(f"Invalid action: {action}")

        self.agent.env['deploy.event'].create({
            'agent_id': self.agent.id,
            'action': action,
            'parameters': parameters or {},
        })

    def deploy(self, branch, is_production):
        return self.queue_action("deploy", {"branch": branch, 'is_production': is_production})

    def undeploy(self, branch):
        return self.queue_action("undeploy", {"branch": branch})

    def backup(self, with_dump):
        return self.queue_action("backup", {"with_dump": with_dump})

    def restore_backup(self, branch):
        return self.queue_action("restore_backup", {"branch": branch})

    def reset_branch(self, branch):
        return self.queue_action("reset_branch", {"branch": branch})

    def update_module(self, branch, module_name):
        return self.queue_action("update_module", {"branch": branch, "module_name": module_name})

    def download_dump(self, is_production):
        return self.queue_action("download_dump", {"is_production": is_production})

    def stream_logs(self, branch):
        return self.queue_action("stream_logs", {"branch": branch})