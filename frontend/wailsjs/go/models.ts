export namespace main {
	
	export class appConfig {
	    canvas_base_url: string;
	    api_token: string;
	    last_course_id: number;
	    course_set: number[];
	
	    static createFrom(source: any = {}) {
	        return new appConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.canvas_base_url = source["canvas_base_url"];
	        this.api_token = source["api_token"];
	        this.last_course_id = source["last_course_id"];
	        this.course_set = source["course_set"];
	    }
	}

}

export namespace models {
	
	export class AnnouncementAuthor {
	    id: number;
	    display_name: string;
	
	    static createFrom(source: any = {}) {
	        return new AnnouncementAuthor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.display_name = source["display_name"];
	    }
	}
	export class Announcement {
	    id: number;
	    title: string;
	    message: string;
	    posted_at: string;
	    delayed_post_at?: string;
	    author: AnnouncementAuthor;
	    read_state: string;
	    discussion_subentry_count: number;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new Announcement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.message = source["message"];
	        this.posted_at = source["posted_at"];
	        this.delayed_post_at = source["delayed_post_at"];
	        this.author = this.convertValues(source["author"], AnnouncementAuthor);
	        this.read_state = source["read_state"];
	        this.discussion_subentry_count = source["discussion_subentry_count"];
	        this.url = source["url"];
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
	
	export class Assignment {
	    id: number;
	    name: string;
	    description: string;
	    due_at?: string;
	    lock_at?: string;
	    unlock_at?: string;
	    points_possible: number;
	    assignment_group_id?: number;
	    submission_types: string[];
	    allowed_attempts?: number;
	    published: boolean;
	    workflow_state: string;
	    has_submitted_submissions: boolean;
	    needs_grading_count: number;
	    html_url: string;
	    position: number;
	
	    static createFrom(source: any = {}) {
	        return new Assignment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.due_at = source["due_at"];
	        this.lock_at = source["lock_at"];
	        this.unlock_at = source["unlock_at"];
	        this.points_possible = source["points_possible"];
	        this.assignment_group_id = source["assignment_group_id"];
	        this.submission_types = source["submission_types"];
	        this.allowed_attempts = source["allowed_attempts"];
	        this.published = source["published"];
	        this.workflow_state = source["workflow_state"];
	        this.has_submitted_submissions = source["has_submitted_submissions"];
	        this.needs_grading_count = source["needs_grading_count"];
	        this.html_url = source["html_url"];
	        this.position = source["position"];
	    }
	}
	export class AssignmentGroup {
	    id: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new AssignmentGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class Attachment {
	    id: number;
	    display_name: string;
	    filename: string;
	    url: string;
	    size: number;
	    "content-type": string;
	
	    static createFrom(source: any = {}) {
	        return new Attachment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.display_name = source["display_name"];
	        this.filename = source["filename"];
	        this.url = source["url"];
	        this.size = source["size"];
	        this["content-type"] = source["content-type"];
	    }
	}
	export class CourseTerm {
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new CourseTerm(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	    }
	}
	export class Course {
	    id: number;
	    name: string;
	    course_code: string;
	    term?: CourseTerm;
	
	    static createFrom(source: any = {}) {
	        return new Course(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.course_code = source["course_code"];
	        this.term = this.convertValues(source["term"], CourseTerm);
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
	export class CourseItem {
	    id: number;
	    name: string;
	    type: string;
	    due_at?: string;
	    points: number;
	    published: boolean;
	    needs_grading_count: number;
	    quiz_type?: string;
	    assignment_id: number;
	    is_old_quiz: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CourseItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.due_at = source["due_at"];
	        this.points = source["points"];
	        this.published = source["published"];
	        this.needs_grading_count = source["needs_grading_count"];
	        this.quiz_type = source["quiz_type"];
	        this.assignment_id = source["assignment_id"];
	        this.is_old_quiz = source["is_old_quiz"];
	    }
	}
	export class CourseStats {
	    student_count: number;
	    total_items: number;
	    published_items: number;
	    needs_grading: number;
	
	    static createFrom(source: any = {}) {
	        return new CourseStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.student_count = source["student_count"];
	        this.total_items = source["total_items"];
	        this.published_items = source["published_items"];
	        this.needs_grading = source["needs_grading"];
	    }
	}
	
	export class FileInfo {
	    id: number;
	    name: string;
	    url: string;
	    download_url: string;
	    public_url: string;
	    folder_path: string;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.download_url = source["download_url"];
	        this.public_url = source["public_url"];
	        this.folder_path = source["folder_path"];
	    }
	}
	export class SubmissionDataItem {
	    question_id: any;
	    text: string;
	    points: number;
	    correct: any;
	
	    static createFrom(source: any = {}) {
	        return new SubmissionDataItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.question_id = source["question_id"];
	        this.text = source["text"];
	        this.points = source["points"];
	        this.correct = source["correct"];
	    }
	}
	export class SubmissionHistoryItem {
	    attachments: Attachment[];
	    submission_data: SubmissionDataItem[];
	
	    static createFrom(source: any = {}) {
	        return new SubmissionHistoryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.attachments = this.convertValues(source["attachments"], Attachment);
	        this.submission_data = this.convertValues(source["submission_data"], SubmissionDataItem);
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
	export class Submission {
	    assignment_id: number;
	    user_id: number;
	    score: any;
	    submitted_at?: string;
	    workflow_state: string;
	    late: boolean;
	    grade?: string;
	    graded_at?: string;
	    grader_id?: number;
	    missing: boolean;
	    excused: boolean;
	    submission_type: string;
	    body: string;
	    url: string;
	    preview_url: string;
	    attempt: number;
	    attachments: Attachment[];
	    submission_history: SubmissionHistoryItem[];
	
	    static createFrom(source: any = {}) {
	        return new Submission(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assignment_id = source["assignment_id"];
	        this.user_id = source["user_id"];
	        this.score = source["score"];
	        this.submitted_at = source["submitted_at"];
	        this.workflow_state = source["workflow_state"];
	        this.late = source["late"];
	        this.grade = source["grade"];
	        this.graded_at = source["graded_at"];
	        this.grader_id = source["grader_id"];
	        this.missing = source["missing"];
	        this.excused = source["excused"];
	        this.submission_type = source["submission_type"];
	        this.body = source["body"];
	        this.url = source["url"];
	        this.preview_url = source["preview_url"];
	        this.attempt = source["attempt"];
	        this.attachments = this.convertValues(source["attachments"], Attachment);
	        this.submission_history = this.convertValues(source["submission_history"], SubmissionHistoryItem);
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
	
	
	export class User {
	    id: number;
	    name: string;
	    email: string;
	    sis_user_id: string;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.email = source["email"];
	        this.sis_user_id = source["sis_user_id"];
	    }
	}

}

