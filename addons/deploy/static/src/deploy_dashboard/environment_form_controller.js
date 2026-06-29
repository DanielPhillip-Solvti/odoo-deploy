/** @odoo-module **/

import {onMounted, onWillDestroy} from "@odoo/owl";
import {FormController} from "@web/views/form/form_controller";
import {formView} from "@web/views/form/form_view";
import {registry} from "@web/core/registry";
import {useService} from "@web/core/utils/hooks";

export class DeployEnvironmentFormController extends FormController {
    setup() {
        super.setup();
        this.busService = useService("bus_service");
        this.notification = useService("notification");
        this.orm = useService("orm");

        onMounted(async () => {
            const record = this.model && this.model.root;
            if (!record || !record.data) return;
            const agentField = record.data.agent_id;
            const agentId = Array.isArray(agentField) ? agentField[0] : agentField;
            if (agentId) {
                await this._initBus(agentId);
            }
        });

        onWillDestroy(() => {
            if (this._busChannel) {
                this.busService.deleteChannel(this._busChannel);
            }
            if (this._heartbeatHandler) {
                this.busService.unsubscribe("deploy.heartbeat", this._heartbeatHandler);
            }
            if (this._callbackHandler) {
                this.busService.unsubscribe("deploy.event_callback", this._callbackHandler);
            }
        });
    }

    async _initBus(agentId) {
        if (!agentId) return;
        this._busChannel = `deploy_agent_${agentId}`;
        this.busService.addChannel(this._busChannel);
        this._heartbeatHandler = (payload) => this._onHeartbeat(payload);
        this.busService.subscribe("deploy.heartbeat", this._heartbeatHandler);
        this._callbackHandler = (payload) => this._onEventCallback(payload);
        this.busService.subscribe("deploy.event_callback", this._callbackHandler);
        this.busService.start();
    }

    async _onHeartbeat() {
        await this.model.root.load();
    }

    async _onEventCallback(payload) {
        const type = payload.status === "success" ? "success" : "danger";
        this.notification.add(payload.message || `Event ${payload.event_id} finished with status ${payload.status}`, {
            type,
            title: payload.status === "success" ? "Complete" : "Failed",
        });
        await this.model.root.load();
    }
}

export const deployEnvironmentFormView = {
    ...formView,
    Controller: DeployEnvironmentFormController,
};

registry.category("views").add("deploy_environment_form", deployEnvironmentFormView);
