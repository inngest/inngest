import { cookies } from 'next/headers';
import { NextRequest, NextResponse } from 'next/server';

export async function POST(request: NextRequest) {
  const { password } = await request.json();

  const expectedPassword = process.env.INNGEST_DEV_DASHBOARD_PASSWORD;

  if (password === expectedPassword) {
    const response = NextResponse.json({ success: true });

    response.cookies.set('inngest_logged_in', 'true', {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'strict',
      maxAge: 60 * 60 * 24, // 24 hours
    });

    return response;
  } else {
    return NextResponse.json({ success: false, error: 'Invalid password' }, { status: 401 });
  }
}
