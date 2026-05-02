import { useState } from 'react';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
import { createFileRoute } from '@tanstack/react-router';

import { useLoginMutation } from '@/store/authApi';

export const Route = createFileRoute('/login')({
  component: LoginPage,
});

function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [login, { isLoading }] = useLoginMutation();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await login({ email, password }).unwrap();
      window.location.href = '/';
    } catch {
      setError('Invalid email or password');
    }
  };

  return (
    <div className="bg-canvasBase flex h-screen w-full items-center justify-center">
      <div className="border-subtle bg-canvasBase w-full max-w-sm rounded-lg border p-8 shadow-lg">
        <div className="mb-8 flex flex-col items-center">
          <InngestLogo className="text-basis mb-2" width={120} />
          <p className="text-muted text-sm">Sign in to your server</p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          {error && (
            <div className="bg-error/10 text-error rounded px-3 py-2 text-sm">
              {error}
            </div>
          )}

          <div className="flex flex-col gap-1.5">
            <label htmlFor="email" className="text-muted text-sm">
              Email
            </label>
            <input
              id="email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="border-subtle bg-canvasSubtle text-basis focus:border-primary-moderate rounded border px-3 py-2 text-sm outline-none"
              placeholder="you@example.com"
              autoComplete="email"
              autoFocus
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <label htmlFor="password" className="text-muted text-sm">
              Password
            </label>
            <input
              id="password"
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="border-subtle bg-canvasSubtle text-basis focus:border-primary-moderate rounded border px-3 py-2 text-sm outline-none"
              placeholder="Enter your password"
              autoComplete="current-password"
            />
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="bg-primary-moderate text-onContrast hover:bg-primary-intense mt-2 rounded px-4 py-2 text-sm font-medium transition-colors disabled:opacity-50"
          >
            {isLoading ? 'Signing in...' : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  );
}
