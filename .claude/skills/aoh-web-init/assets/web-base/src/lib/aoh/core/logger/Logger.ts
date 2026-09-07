import { get, readable, type Readable } from 'svelte/store';
import { logger as aohLogger, type AohLogger } from '@mssfoobar/logger';

const pinoLogger: Readable<AohLogger> = readable(aohLogger);
export const log = get(pinoLogger);
