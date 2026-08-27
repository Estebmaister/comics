import { FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import './Auth.css';

export const LoginPage = () => {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setError(null);
    const message = await login(email, password);
    if (message) {
      setError(message);
      return;
    }
    navigate('/');
  };

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <h1>Sign in</h1>
        <p>Browse comics without signing in. Use an account to manage your profile.</p>
        {error ? <div className="auth-error">{error}</div> : null}
        <form className="auth-form" onSubmit={onSubmit}>
          <label>
            Email
            <input type="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
          </label>
          <label>
            Password
            <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} required />
          </label>
          <div className="auth-actions">
            <button type="submit" className="auth-button">Sign in</button>
            <Link className="auth-link" to="/signup">Create account</Link>
          </div>
        </form>
      </div>
    </div>
  );
};
