export type BroadcastMessage<T = unknown> = {
	type: string;
	payload?: T;
};

export function createBroadcaster<T = unknown>(channelName: string) {
	if (typeof BroadcastChannel === 'undefined') {
		return {
			post: (_msg: BroadcastMessage<T>) => {},
			subscribe: (_handler: (msg: BroadcastMessage<T>) => void) => () => {},
			close: () => {}
		};
	}
	const channel = new BroadcastChannel(channelName);
	return {
		post(msg: BroadcastMessage<T>) {
			channel.postMessage(msg);
		},
		subscribe(handler: (msg: BroadcastMessage<T>) => void) {
			const listener = (ev: MessageEvent<BroadcastMessage<T>>) => handler(ev.data);
			channel.addEventListener('message', listener);
			return () => channel.removeEventListener('message', listener);
		},
		close() {
			channel.close();
		}
	};
}
