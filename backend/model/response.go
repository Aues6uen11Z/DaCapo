package model

type Status struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var (
	StatusSuccess = Status{Code: 0, Message: "Success"}

	StatusFile      = Status{Code: 1001, Message: "File operation failed"}
	StatusDatabase  = Status{Code: 1002, Message: "Database error"}
	StatusDuplicate = Status{Code: 1003, Message: "Instance name already exists"}
	StatusGit       = Status{Code: 1004, Message: "Git repository operation failed"}
	StatusPython    = Status{Code: 1005, Message: "Python environment operation failed"}
)

type RspGetInstance struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`

	WorkingTemplate []string        `json:"working_template"`
	Ready           map[string]bool `json:"ready"`
	Layout          any             `json:"layout"`
	Translation     any             `json:"translation"`
}

type RspGetTemplate struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`

	Templates []string `json:"templates"`
}

type RspUpdateRepo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`

	IsUpdated bool `json:"is_updated"`
}

type RspTaskQueue struct {
	Type         string    `json:"type"`
	InstanceName string    `json:"instance_name"`
	Queue        TaskQueue `json:"queue"`
}

type RspLogMessage struct {
	Type         string `json:"type"`
	InstanceName string `json:"instance_name"`
	Content      string `json:"content"`
}

type RspSchedulerState struct {
	Type         string `json:"type"`
	InstanceName string `json:"instance_name"`
	State        string `json:"state"`
}

// Settings response
type RspSettings struct {
	Language          string `json:"language"`
	RunOnStartup      bool   `json:"runOnStartup"`
	SchedulerCron     string `json:"schedulerCron"`
	AutoActionTrigger string `json:"autoActionTrigger"`
	AutoActionCron    string `json:"autoActionCron"`
	AutoActionType    string `json:"autoActionType"`
}
