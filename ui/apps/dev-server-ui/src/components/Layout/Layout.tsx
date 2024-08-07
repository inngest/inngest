'use client';

import React, { useEffect, useState, type ReactNode } from 'react';

import SideBar from './SideBar';

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="flex w-full flex-row justify-start">
      <SideBar />

      <div className="flex w-full flex-col">{children}</div>
    </div>
  );
}
