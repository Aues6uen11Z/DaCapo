export interface ReqFromLocal {
    instance_name: string;
    template_name: string;
    template_path: string;
}

export interface ReqFromTemplate {
    instance_name: string;
    template_name: string;
}

export interface ReqFromRemote {
    instance_name: string;
    template_name: string;
    url: string;
    local_path: string;
    template_rel_path: string;
    branch: string;
}

export interface ReqUpdateInstance {
    menu: string;
    task: string;
    group: string;
    item: string;
    value: unknown;
}
