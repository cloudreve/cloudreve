import React from "react";

function ArrowLeft(props: React.HTMLAttributes<SVGElement>) {
  return (
    <svg width="44" height="44" viewBox="0 0 768 768" {...props}>
      <path d="M640.5 352.5v63h-390l178.5 180-45 45-256.5-256.5 256.5-256.5 45 45-178.5 180h390z" />
    </svg>
  );
}

export default ArrowLeft;
