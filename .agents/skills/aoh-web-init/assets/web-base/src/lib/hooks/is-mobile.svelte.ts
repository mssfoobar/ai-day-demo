const MOBILE_BREAKPOINT = 768;

export class IsMobile {
	current = $state(false);

	constructor() {
		$effect(() => {
			const mql = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);
			const onChange = () => {
				this.current = mql.matches;
			};
			mql.addEventListener('change', onChange);
			this.current = mql.matches;
			return () => mql.removeEventListener('change', onChange);
		});
	}
}
