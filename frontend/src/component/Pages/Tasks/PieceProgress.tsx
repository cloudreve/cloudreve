import { memo, useEffect, useRef } from "react";
import { Box, useTheme } from "@mui/material";

export interface PieceProgressProps {
  pieces: string;
  total: number;
}

const PieceProgress: React.FC<PieceProgressProps> = memo(({ pieces, total }) => {
  const theme = useTheme();
  const canvasRef = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    if (!pieces || !canvasRef.current || !total) {
      return;
    }

    const canvas = canvasRef.current;
    const context = canvas.getContext("2d");
    if (!context) {
      return;
    }

    context.clearRect(0, 0, canvas.width, canvas.height);
    context.strokeStyle = theme.palette.primary.light;

    const bits = Uint8Array.from(atob(pieces), (c) => c.charCodeAt(0));
    for (let i = 0; i < canvas.width; i++) {
      let bitIndex = Math.floor(((i + 1) / canvas.width) * total);
      // Read bool from unit8 array
      let bit = bits[Math.floor(bitIndex / 8)] & (1 << bitIndex % 8);
      if (bit) {
        context.beginPath();
        context.moveTo(i, 0);
        context.lineTo(i, canvas.height);
        context.stroke();
      }
    }
  }, [pieces, theme, total]);
  return (
    <Box
      sx={{
        mt: 2,
        width: "100%",
        height: "36px",
        borderRadius: "12px",
        backgroundColor: (theme) => theme.palette.action.hover,
      }}
    >
      <canvas
        ref={canvasRef}
        width={1400}
        height={200}
        style={{
          borderRadius: "12px",
          width: "100%",
          height: "100%",
        }}
      />
    </Box>
  );
});

export default PieceProgress;
