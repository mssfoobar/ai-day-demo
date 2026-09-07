import { writable } from 'svelte/store';

export type BreadcrumbItem = { name: string; url?: string };

export const breadcrumb = writable<BreadcrumbItem[]>([]);

export function setBreadcrumb(items: BreadcrumbItem[]) {
	breadcrumb.set(items);
}

export function clearBreadcrumb() {
	breadcrumb.set([]);
}
