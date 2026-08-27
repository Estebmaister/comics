import { FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import './Auth.css';

export const SignupPage = () => {
  const { signup } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setError(null);
    const message = await signup(email, username, password);
    if (message) {
      setError(message);
      return;
    }
    navigate('/');
  };

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <h1>Create account</h1>
        <p>Comics stay public. Accounts unlock profile management on the Go backend.</p>
        {error ? <div className="auth-error">{error}</div> : null}
        <form className="auth-form" onSubmit={onSubmit}>
          <label>
            Email
            <input type="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
          </label>
          <label>
            Username
            <input value={username} onChange={(event) => setUsername(event.target.value)} required />
          </label>
          <label>
            Password
            <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} required />
          </label>
          <div className="auth-actions">
            <button type="submit" className="auth-button">Sign up</button>
            <Link className="auth-link" to="/login">Back to sign in</Link>
          </div>
        </form>
      </div>
    </div>
  );
};
