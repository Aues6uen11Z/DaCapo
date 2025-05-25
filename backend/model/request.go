package model

// ReqFromLocal represents a request to create a new instance from a local disk template
type ReqFromLocal struct {
	InstanceName string `json:"instance_name" binding:"required"`
	TemplateName string `json:"template_name" binding:"required"`
	TemplatePath string `json:"template_path" binding:"required"`
}

// ReqFromTemplate represents a request to create a new instance from an existing template
type ReqFromTemplate struct {
	InstanceName string `json:"instance_name" binding:"required"`
	TemplateName string `json:"template_name" binding:"required"`
}

// ReqFromRemote represents a request to create a new instance from a remote git repository
type ReqFromRemote struct {
	InstanceName    string `json:"instance_name" binding:"required"`
	TemplateName    string `json:"template_name" binding:"required"`
	URL             string `json:"url" binding:"required"`
	Branch          string `json:"branch"`
	LocalPath       string `json:"local_path" binding:"required"`
	TemplateRelPath string `json:"template_rel_path" binding:"required"`
}

// ReqUpdateInstance represents a request to update instance configuration
type ReqUpdateInstance struct {
	Menu  string `json:"menu" binding:"required"`
	Task  string `json:"task" binding:"required"`
	Group string `json:"group" binding:"required"`
	Item  string `json:"item" binding:"required"`
	Value any    `json:"value" binding:"required"`
}

// ReqUpdateQueue represents a request to update the homepage task queue
type ReqUpdateQueue struct {
	Queues map[string]TaskQueue `json:"queues" binding:"required"`
}

// ReqSchedulerState represents a request to start or stop task execution (type: "start" / "stop")
type ReqSchedulerState struct {
	Type         string `json:"type" binding:"required"`
	InstanceName string `json:"instance_name"`
	AutoClose    bool   `json:"auto_close"`
}

type ReqSchedulerCron struct {
	CronExpr string `json:"cron_expr" binding:"required"`
}

// Settings related requests
type ReqUpdateSettings struct {
	Language          string  `json:"language"`
	RunOnStartup      *bool   `json:"runOnStartup"`
	SchedulerCron     *string `json:"schedulerCron"`
	AutoActionTrigger string  `json:"autoActionTrigger"`
	AutoActionCron    *string `json:"autoActionCron"`
	AutoActionType    string  `json:"autoActionType"`
}
