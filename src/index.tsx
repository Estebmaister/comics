import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';

import { AuthProvider } from './frontend/context/AuthContext';
import { LoginPage } from './frontend/components/Auth/LoginPage';
import { SignupPage } from './frontend/components/Auth/SignupPage';
import { ProfilePage } from './frontend/components/Auth/ProfilePage';
import { ComicsMainPage } from './frontend/components/Comics/MainPage/MainPage';
import { appBasename } from './frontend/util/RouterBasename';
import reportWebVitals from './frontend/reportWebVitals';

const rootElement = document.getElementById('root') as HTMLElement;
const root = createRoot(rootElement);
root.render(
  <StrictMode>
    <Router basename={appBasename}>
      <AuthProvider>
        <Routes>
          <Route path="/" element={<ComicsMainPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/signup" element={<SignupPage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Routes>
      </AuthProvider>
    </Router>
  </StrictMode>
);

if (import.meta.env.DEV) {
  reportWebVitals(console.log);
}
