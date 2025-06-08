import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/api': {
				target: 'http://localhost:8081',
				changeOrigin: true
			},
			'/ws': {
				target: 'ws://localhost:8081',
				ws: true,
				changeOrigin: true
			}
		}
	}
});
