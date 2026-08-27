import { FormEvent, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { updateProfile } from '../../util/AuthHelpers';
import './Auth.css';

export const ProfilePage = () => {
  const { user, accessToken, refreshProfile, logout, isAuthenticated } = useAuth();
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [message, setMessage] = useState<string | null>(null);

  useEffect(() => {
    void refreshProfile();
  }, [refreshProfile]);

  useEffect(() => {
    setEmail(user?.email ?? '');
    setUsername(user?.username ?? '');
  }, [user]);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setMessage(null);
    const response = await updateProfile(
      { email, username, password: password || undefined },
      accessToken,
    );
    if (response.status !== 200) {
      setMessage(response.message ?? 'Profile update failed');
      return;
    }
    setPassword('');
    setMessage('Profile updated');
    await refreshProfile();
  };

  if (!isAuthenticated) {
    return (
      <div className="auth-shell">
        <div className="auth-card">
          <h1>Profile</h1>
          <p>Sign in to view and edit your account.</p>
          <Link className="auth-link" to="/login">Go to sign in</Link>
        </div>
      </div>
    );
  }

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <h1>Profile</h1>
        <p>Signed in as {user?.username ?? user?.email ?? 'reader'}.</p>
        {message ? <div className="auth-error">{message}</div> : null}
        <form className="auth-form" onSubmit={onSubmit}>
          <label>
            Email
            <input type="email" value={email} onChange={(event) => setEmail(event.target.value)} />
          </label>
          <label>
            Username
            <input value={username} onChange={(event) => setUsername(event.target.value)} />
          </label>
          <label>
            New password
            <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
          </label>
          <div className="auth-actions">
            <button type="submit" className="auth-button">Save profile</button>
            <button type="button" className="auth-link" onClick={logout}>Sign out</button>
            <Link className="auth-link" to="/">Back to comics</Link>
          </div>
        </form>
      </div>
    </div>
  );
};
