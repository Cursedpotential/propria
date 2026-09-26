export namespace main {
	
	export class CloudDrive {
	    type: string;
	    provider: string;
	    mount_point: string;
	    path: string;
	    email: string;
	    mounted: boolean;
	    display_name: string;
	    detected: boolean;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new CloudDrive(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.provider = source["provider"];
	        this.mount_point = source["mount_point"];
	        this.path = source["path"];
	        this.email = source["email"];
	        this.mounted = source["mounted"];
	        this.display_name = source["display_name"];
	        this.detected = source["detected"];
	        this.notes = source["notes"];
	    }
	}
	export class FileNode {
	    id: string;
	    name: string;
	    path: string;
	    size: number;
	    formattedSize: string;
	    created: time.Time;
	    modified: time.Time;
	    accessed: time.Time;
	    isDir: boolean;
	    isSymlink: boolean;
	    isHidden: boolean;
	    isSystem: boolean;
	    isReadOnly: boolean;
	    isCloud: boolean;
	    cloudType?: string;
	    extension: string;
	    mimeType: string;
	    permissions: string;
	    owner: string;
	    hash?: string;
	    attributes: number;
	    childCount: number;
	    childrenLoaded: boolean;
	    children?: FileNode[];
	    parentId?: string;
	
	    static createFrom(source: any = {}) {
	        return new FileNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.formattedSize = source["formattedSize"];
	        this.created = this.convertValues(source["created"], time.Time);
	        this.modified = this.convertValues(source["modified"], time.Time);
	        this.accessed = this.convertValues(source["accessed"], time.Time);
	        this.isDir = source["isDir"];
	        this.isSymlink = source["isSymlink"];
	        this.isHidden = source["isHidden"];
	        this.isSystem = source["isSystem"];
	        this.isReadOnly = source["isReadOnly"];
	        this.isCloud = source["isCloud"];
	        this.cloudType = source["cloudType"];
	        this.extension = source["extension"];
	        this.mimeType = source["mimeType"];
	        this.permissions = source["permissions"];
	        this.owner = source["owner"];
	        this.hash = source["hash"];
	        this.attributes = source["attributes"];
	        this.childCount = source["childCount"];
	        this.childrenLoaded = source["childrenLoaded"];
	        this.children = this.convertValues(source["children"], FileNode);
	        this.parentId = source["parentId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScanOptions {
	    path: string;
	    maxDepth: number;
	    includeHidden: boolean;
	    includeSystem: boolean;
	    followSymlinks: boolean;
	    minSize: number;
	    maxSize: number;
	    extensions: string[];
	    excludeDirs: string[];
	    excludePatterns: string[];
	    threadCount: number;
	    calculateHash: boolean;
	    cacheResults: boolean;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.maxDepth = source["maxDepth"];
	        this.includeHidden = source["includeHidden"];
	        this.includeSystem = source["includeSystem"];
	        this.followSymlinks = source["followSymlinks"];
	        this.minSize = source["minSize"];
	        this.maxSize = source["maxSize"];
	        this.extensions = source["extensions"];
	        this.excludeDirs = source["excludeDirs"];
	        this.excludePatterns = source["excludePatterns"];
	        this.threadCount = source["threadCount"];
	        this.calculateHash = source["calculateHash"];
	        this.cacheResults = source["cacheResults"];
	        this.pageSize = source["pageSize"];
	    }
	}
	export class ScanProgress {
	    sessionId: string;
	    currentPath: string;
	    filesScanned: number;
	    dirsScanned: number;
	    totalSize: number;
	    formattedSize: string;
	    errors: string[];
	    percentage: number;
	    speed: number;
	    timeRemaining: number;
	    currentDepth: number;
	    estimatedTotal: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.currentPath = source["currentPath"];
	        this.filesScanned = source["filesScanned"];
	        this.dirsScanned = source["dirsScanned"];
	        this.totalSize = source["totalSize"];
	        this.formattedSize = source["formattedSize"];
	        this.errors = source["errors"];
	        this.percentage = source["percentage"];
	        this.speed = source["speed"];
	        this.timeRemaining = source["timeRemaining"];
	        this.currentDepth = source["currentDepth"];
	        this.estimatedTotal = source["estimatedTotal"];
	        this.status = source["status"];
	    }
	}
	export class ScanStatus {
	    running: boolean;
	    files_scanned: number;
	    dirs_scanned: number;
	    current_path: string;
	    start_time: time.Time;
	    elapsed_seconds: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.files_scanned = source["files_scanned"];
	        this.dirs_scanned = source["dirs_scanned"];
	        this.current_path = source["current_path"];
	        this.start_time = this.convertValues(source["start_time"], time.Time);
	        this.elapsed_seconds = source["elapsed_seconds"];
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Settings {
	    threadCount: number;
	    cacheResults: boolean;
	    defaultExportFormat: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.threadCount = source["threadCount"];
	        this.cacheResults = source["cacheResults"];
	        this.defaultExportFormat = source["defaultExportFormat"];
	    }
	}

}

export namespace time {
	
	export class Time {
	
	
	    static createFrom(source: any = {}) {
	        return new Time(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

