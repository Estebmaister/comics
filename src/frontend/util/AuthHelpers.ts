import config from './Config';

const SERVER = config.SERVER;

export type AuthData = {
  user_id: string;
  access_token?: string;
  refresh_token?: string;
};

export type AuthResponse = {
  status: number;
  data?: AuthData;
  message?: string;
};

export type AuthUser = {
  id?: string;
  username?: string;
  email?: string;
};

const authFetch = async (path: string, init: RequestInit = {}): Promise<Response> => {
  const headers = new Headers(init.headers ?? {});
  if (!headers.has('Content-Type') && init.body) {
    headers.set('Content-Type', 'application/json');
  }
  headers.set('Accept', 'application/json');
  return fetch(`${SERVER}${path}`, {
    ...init,
    headers,
    credentials: 'include',
  });
};

export const signup = async (payload: {
  email: string;
  username: string;
  password: string;
}): Promise<AuthResponse> => {
  const response = await authFetch('/signup', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  return response.json();
};

export const login = async (payload: {
  email: string;
  password: string;
}): Promise<AuthResponse> => {
  const response = await authFetch('/login', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  return response.json();
};

export const refreshAuth = async (): Promise<AuthResponse> => {
  const response = await authFetch('/refresh-token', {
    method: 'POST',
    headers: {
      Role: 'user',
    },
  });
  return response.json();
};

export const fetchProfile = async (accessToken?: string): Promise<AuthUser> => {
  const headers: Record<string, string> = {};
  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`;
  }
  const response = await authFetch('/protected/profile', { headers });
  if (!response.ok) {
    throw new Error('Profile unavailable');
  }
  return response.json();
};

export const updateProfile = async (
  payload: { email?: string; username?: string; password?: string },
  accessToken?: string,
): Promise<AuthResponse> => {
  const headers: Record<string, string> = {};
  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`;
  }
  const response = await authFetch('/protected/profile', {
    method: 'PUT',
    headers,
    body: JSON.stringify(payload),
  });
  return response.json();
};

export const storeAccessToken = (token?: string) => {
  if (!token) {
    sessionStorage.removeItem('comics_access_token');
    return;
  }
  sessionStorage.setItem('comics_access_token', token);
};

export const readAccessToken = () => sessionStorage.getItem('comics_access_token') ?? undefined;

export const clearAuthStorage = () => {
  sessionStorage.removeItem('comics_access_token');
};
