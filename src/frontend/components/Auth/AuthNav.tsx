import { Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import './Auth.css';

export const AuthNav = () => {
  const { isAuthenticated, user, logout } = useAuth();

  return (
    <div className="auth-nav">
      <Link to="/">Comics</Link>
      {isAuthenticated ? (
        <>
          <Link to="/profile">{user?.username ?? 'Profile'}</Link>
          <button type="button" onClick={logout}>Sign out</button>
        </>
      ) : (
        <>
          <Link to="/login">Sign in</Link>
          <Link to="/signup">Sign up</Link>
        </>
      )}
    </div>
  );
};
