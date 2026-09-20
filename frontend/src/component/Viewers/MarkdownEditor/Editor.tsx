import {
  AdmonitionDirectiveDescriptor,
  BlockTypeSelect,
  BoldItalicUnderlineToggles,
  ChangeAdmonitionType,
  ChangeCodeMirrorLanguage,
  codeBlockPlugin,
  CodeMirrorEditor,
  codeMirrorPlugin,
  CodeToggle,
  ConditionalContents,
  CreateLink,
  diffSourcePlugin,
  DiffSourceToggleWrapper,
  DirectiveNode,
  directivesPlugin,
  EditorInFocus,
  frontmatterPlugin,
  headingsPlugin,
  imagePlugin,
  InsertAdmonition,
  InsertCodeBlock,
  InsertFrontmatter,
  InsertImage,
  InsertTable,
  InsertThematicBreak,
  linkDialogPlugin,
  linkPlugin,
  listsPlugin,
  ListsToggle,
  markdownShortcutPlugin,
  MDXEditor,
  quotePlugin,
  Separator,
  ShowSandpackInfo,
  StrikeThroughSupSubToggles,
  tablePlugin,
  thematicBreakPlugin,
  toolbarPlugin,
  UndoRedo,
} from "@mdxeditor/editor";
import "@mdxeditor/editor/style.css";
import { Box } from "@mui/material";
import i18next from "i18next";
import { memo, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import "./editor.css";

export interface MarkdownEditorProps {
  value: string;
  darkMode?: boolean;
  initialValue: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
  displayOnly?: boolean;
  onSaveShortcut?: () => void;
  imageAutocompleteSuggestions?: string[] | null;
  imagePreviewHandler?: (imageSource: string) => Promise<string>;
  imageUploadHandler?: ((image: File) => Promise<string>) | null;
}

function whenInAdmonition(editorInFocus: EditorInFocus | null) {
  const node = editorInFocus?.rootNode;
  if (!node || node.getType() !== "directive") {
    return false;
  }

  return ["note", "tip", "danger", "info", "caution"].includes((node as DirectiveNode).getMdastNode().name);
}

function setEndOfContenteditable(contentEditableElement: HTMLElement) {
  let range: Range | null;
  let selection: Selection | null;
  if (document.createRange) {
    range = document.createRange(); //Create a range (a range is a like the selection but invisible)
    range.selectNodeContents(contentEditableElement); //Select the entire contents of the element with the range
    range.collapse(false); //collapse the range to the end point. false means collapse to end rather than the start
    selection = window.getSelection(); //get the selection object (allows you to change selection)
    selection?.removeAllRanges(); //remove any selections already made
    selection?.addRange(range); //make the range you have just created the visible selection
  }
}

const MarkdownEditor = (props: MarkdownEditorProps) => {
  const { t } = useTranslation();
  const [nsLoaded, setNsLoaded] = useState(false);
  useEffect(() => {
    i18next.loadNamespaces(["markdown_editor"]).then(() => {
      setNsLoaded(true);
    });
  }, []);

  // MDXEditor invokes plugin.update() for every plugin on every render;
  // keeping the array referentially stable lets the realm suppress
  // downstream updates for unchanged params.
  const plugins = useMemo(
    () => [
            diffSourcePlugin({
              diffMarkdown: props.initialValue,
              viewMode: "rich-text",
            }),
            ...(props.displayOnly
              ? []
              : [
                  toolbarPlugin({
                    toolbarContents: () => (
                      <ConditionalContents
                        options={[
                          {
                            when: (editor) => editor?.editorType === "codeblock",
                            contents: () => <ChangeCodeMirrorLanguage />,
                          },
                          {
                            when: (editor) => editor?.editorType === "sandpack",
                            contents: () => <ShowSandpackInfo />,
                          },
                          {
                            fallback: () => (
                              <DiffSourceToggleWrapper>
                                <UndoRedo />
                                <Separator />
                                <BoldItalicUnderlineToggles />
                                <CodeToggle />
                                <Separator />
                                <StrikeThroughSupSubToggles />
                                <Separator />
                                <ListsToggle />
                                <Separator />

                                <ConditionalContents
                                  options={[
                                    {
                                      when: whenInAdmonition,
                                      contents: () => <ChangeAdmonitionType />,
                                    },
                                    { fallback: () => <BlockTypeSelect /> },
                                  ]}
                                />

                                <Separator />

                                <CreateLink />
                                <InsertImage />

                                <Separator />

                                <InsertTable />
                                <InsertThematicBreak />

                                <Separator />
                                <InsertCodeBlock />

                                <ConditionalContents
                                  options={[
                                    {
                                      when: (editorInFocus) => !whenInAdmonition(editorInFocus),
                                      contents: () => (
                                        <>
                                          <Separator />
                                          <InsertAdmonition />
                                        </>
                                      ),
                                    },
                                  ]}
                                />

                                <Separator />
                                <InsertFrontmatter />
                              </DiffSourceToggleWrapper>
                            ),
                          },
                        ]}
                      />
                    ),
                  }),
                ]),
            listsPlugin(),
            quotePlugin(),
            headingsPlugin({ allowedHeadingLevels: [1, 2, 3] }),
            linkPlugin(),
            linkDialogPlugin(),
            imagePlugin({
              imageUploadHandler: props.imageUploadHandler,
              imagePreviewHandler: props.imagePreviewHandler,
              imageAutocompleteSuggestions: props.imageAutocompleteSuggestions ?? undefined,
            }),
            tablePlugin(),
            thematicBreakPlugin(),
            frontmatterPlugin(),
            codeBlockPlugin({
              defaultCodeBlockLanguage: "",
              codeBlockEditorDescriptors: [{ priority: -10, match: (_) => true, Editor: CodeMirrorEditor }],
            }),
            codeMirrorPlugin({
              codeBlockLanguages: {
                js: "JavaScript",
                jsx: "JSX",
                css: "CSS",
                txt: "Plain Text",
                tsx: "TSX",
                ts: "TypeScript",
                html: "HTML",
                json: "JSON",
                sh: "Shell",
                bash: "Bash",
                yaml: "YAML",
                markdown: "Markdown",
                dockerfile: "Dockerfile",
                sql: "SQL",
                python: "Python",
                go: "Go",
                java: "Java",
                c: "C",
                cpp: "C++",
                php: "PHP",
                ruby: "Ruby",
                perl: "Perl",
                swift: "Swift",
                r: "R",
                rust: "Rust",
                kotlin: "Kotlin",
                scala: "Scala",
                "": "Unspecified",
              },
            }),
            directivesPlugin({
              directiveDescriptors: [AdmonitionDirectiveDescriptor],
            }),
            markdownShortcutPlugin(),
          ],
    [
      props.displayOnly,
      props.initialValue,
      props.imageUploadHandler,
      props.imagePreviewHandler,
      props.imageAutocompleteSuggestions,
    ],
  );

  const translation = useCallback(
    (key: string, _defaultValue: string, interpolations?: Record<string, unknown>) =>
      t("markdown_editor:" + key, interpolations),
    [t],
  );

  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        minHeight: props.displayOnly ? "100%" : "calc(100vh - 200px)",
      }}
      onKeyDown={(e) => {
        if ((e.ctrlKey || e.metaKey) && e.key === "s") {
          e.preventDefault();
          props.onSaveShortcut?.();
        }
      }}
    >
      {nsLoaded && (
        <MDXEditor
          className={props.darkMode ? "dark-theme dark-editor" : undefined}
          translation={translation}
          readOnly={props.readOnly}
          onChange={props.onChange}
          plugins={plugins}
          contentEditableClassName={props.darkMode ? "markdown-body-dark" : "markdown-body-light"}
          markdown={props.value}
        />
      )}
      {!nsLoaded && <div className={"mdxeditor"}></div>}
      <Box
        onClick={() => {
          setEndOfContenteditable(
            document.querySelector(props.darkMode ? ".markdown-body-dark" : ".markdown-body-light") as HTMLElement,
          );
        }}
        sx={{
          cursor: "text",
          flexGrow: 1,
        }}
      />
    </Box>
  );
};

// Memoized: the parent re-renders on every keystroke (changedValue state);
// re-rendering MDXEditor re-runs every plugin's update() hook, which is the
// dominant source of typing lag on large documents.
export default memo(MarkdownEditor);
