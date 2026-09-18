import { useDrop } from "react-dnd";
import { useEffect } from "react";

const DisableDropDelay = () => {
  const [_, bodyDropRef] = useDrop(() => ({
    accept: "file",
    drop: () => {
      // do something
    },
  }));

  useEffect(() => {
    bodyDropRef(document.body);
    return () => {
      bodyDropRef(null);
    };
  }, []);
};

export default DisableDropDelay;
