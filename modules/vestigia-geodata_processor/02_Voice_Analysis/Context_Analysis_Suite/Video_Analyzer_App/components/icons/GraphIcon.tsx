
import React from 'react';

export const GraphIcon: React.FC<React.SVGProps<SVGSVGElement>> = (props) => (
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
    <circle cx="5.5" cy="5.5" r="3.5"/>
    <circle cx="18.5" cy="5.5" r="3.5"/>
    <circle cx="5.5" cy="18.5" r="3.5"/>
    <circle cx="18.5" cy="18.5" r="3.5"/>
    <line x1="5.5" y1="9" x2="5.5" y2="15"/>
    <line x1="18.5" y1="9" x2="18.5" y2="15"/>
    <line x1="9" y1="5.5" x2="15" y2="5.5"/>
    <line x1="9" y1="18.5" x2="15" y2="18.5"/>
  </svg>
);
