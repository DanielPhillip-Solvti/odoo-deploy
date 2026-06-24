package config

var AgentPathAgent string = "/data/deploy-agent"

func EnvPathAgent(branch string) string {
	return AgentPathAgent + "/envs/" + branch
}

func AddonsPathAgent(branch string) string {
	return EnvPathAgent(branch) + "/addons"
}

var AddonsPathContainer string = "/code/addons"
