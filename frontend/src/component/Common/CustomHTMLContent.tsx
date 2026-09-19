import { useEffect, useRef } from "react";

export interface CustomHTMLContentProps {
  html: string;
}

// CustomHTMLContent renders administrator-supplied HTML. Scripts inside the
// markup do not run when injected via innerHTML, so each <script> element is
// re-created as a fresh node — dynamically inserted scripts execute normally.
// The content is trusted by definition (only admins can configure it).
const CustomHTMLContent = ({ html }: CustomHTMLContentProps) => {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = ref.current;
    if (!container) {
      return;
    }

    container.querySelectorAll("script").forEach((oldScript) => {
      const script = document.createElement("script");
      Array.from(oldScript.attributes).forEach((attr) => {
        script.setAttribute(attr.name, attr.value);
      });
      script.textContent = oldScript.textContent;
      oldScript.replaceWith(script);
    });
  }, [html]);

  return <div ref={ref} dangerouslySetInnerHTML={{ __html: html }} />;
};

export default CustomHTMLContent;
