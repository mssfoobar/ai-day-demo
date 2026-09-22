/**
 * The map does not server-render.
 *
 * The Cesium engine touches browser globals (`window`, `document`, WebGL) at module
 * initialisation, so importing it on the server throws before anything renders. Opting
 * the route out is what keeps the server response free of engine markup and the server
 * log free of WebGL errors.
 */
export const ssr = false;
