import React from 'react';
import { createRoot } from 'react-dom/client';
import { Trace, clipTo } from './Trace.jsx';
import { eventsFromTrace } from './fromPayload.js';

/**
 * What the page gets: the renderer, as a component, plus the one function that
 * knows how to turn a raw Dev Server payload into the events it takes.
 */
window.TraceKit = { React, createRoot, Trace, clipTo, eventsFromTrace };
