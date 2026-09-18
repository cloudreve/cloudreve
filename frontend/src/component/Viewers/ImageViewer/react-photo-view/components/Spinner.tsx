import React from "react";
import "./Spinner.less";

// eslint-disable-next-line react/prop-types
function Spinner({ className = "", ...props }: React.HTMLAttributes<HTMLElement>) {
  return (
    <div className={`PhotoView__Spinner ${className}`} {...props}>
      <svg viewBox="0 0 32 32" width="36" height="36" fill="white">
        <path opacity=".25" d="M16 0 A16 16 0 0 0 16 32 A16 16 0 0 0 16 0 M16 4 A12 12 0 0 1 16 28 A12 12 0 0 1 16 4" />
        <path d="M16 0 A16 16 0 0 1 32 16 L28 16 A12 12 0 0 0 16 4z" />
      </svg>
    </div>
  );
}

export default Spinner;
