import { createConnectTransport } from '@connectrpc/connect-web';
import type { Interceptor } from '@connectrpc/connect';

// createAuthInterceptor creates an interceptor that adds JWT to requests.
const createAuthInterceptor = (): Interceptor => {
  return (next) => async (req) => {
    const token = localStorage.getItem('cops_access_token');
    if (token && token.length > 0) {
      req.header.set('Authorization', `Bearer ${token}`);
    }
    return await next(req);
  };
};

export const transport = createConnectTransport({
  baseUrl: import.meta.env.VITE_API_URL || 'http://localhost:8080',
  interceptors: [createAuthInterceptor()],
});
