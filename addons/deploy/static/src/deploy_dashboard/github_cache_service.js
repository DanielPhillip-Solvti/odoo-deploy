/** @odoo-module **/
/* global Map */

import {registry} from "@web/core/registry";

const cache = new Map();

function key(agentId, branch) {
    return `${agentId}:${branch}`;
}

export const deployGithubCache = {
    dependencies: [],
    start() {
        return {
            get(agentId, branch) {
                return cache.get(key(agentId, branch)) || null;
            },
            set(agentId, branch, data) {
                cache.set(key(agentId, branch), data);
            },
            clear() {
                cache.clear();
            },
        };
    },
};

registry.category("services").add("deploy_github_cache", deployGithubCache);
