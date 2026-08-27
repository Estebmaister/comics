import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import {
  clearAuthStorage,
  fetchProfile,
  login as loginRequest,
  readAccessToken,
  signup as signupRequest,
  storeAccessToken,
  type AuthUser,
} from '../util/AuthHelpers';

type AuthContextValue = {
  user: AuthUser | null;
  accessToken?: string;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<string | null>;
  signup: (email: string, username: string, password: string) => Promise<string | null>;
  logout: () => void;
  refreshProfile: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [accessToken, setAccessToken] = useState<string | undefined>(readAccessToken());

  const refreshProfile = useCallback(async () => {
    const token = readAccessToken();
    if (!token) {
      setUser(null);
      return;
    }
    try {
      const profile = await fetchProfile(token);
      setUser(profile);
      setAccessToken(token);
    } catch {
      clearAuthStorage();
      setUser(null);
      setAccessToken(undefined);
    }
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const response = await loginRequest({ email, password });
    if (response.status !== 200 || !response.data?.access_token) {
      return response.message ?? 'Login failed';
    }
    storeAccessToken(response.data.access_token);
    setAccessToken(response.data.access_token);
    await refreshProfile();
    return null;
  }, [refreshProfile]);

  const signup = useCallback(async (email: string, username: string, password: string) => {
    const response = await signupRequest({ email, username, password });
    if (response.status !== 201) {
      return response.message ?? 'Signup failed';
    }
    return login(email, password);
  }, [login]);

  const logout = useCallback(() => {
    clearAuthStorage();
    setUser(null);
    setAccessToken(undefined);
  }, []);

  const value = useMemo(
    () => ({
      user,
      accessToken,
      isAuthenticated: Boolean(accessToken),
      login,
      signup,
      logout,
      refreshProfile,
    }),
    [accessToken, login, logout, refreshProfile, signup, user],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
};
