import { Box, IconButton, Stack, Table, TableBody, TableContainer, TableHead, TableRow } from "@mui/material";
import React, { memo, useMemo, useState } from "react";
import { DndProvider, useDrag, useDrop } from "react-dnd";
import { HTML5Backend } from "react-dnd-html5-backend";
import { useTranslation } from "react-i18next";
import {
  DenseFilledTextField,
  NoWrapCell,
  NoWrapTableCell,
  SecondaryButton,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents.tsx";
import Add from "../../../Icons/Add.tsx";
import ArrowDown from "../../../Icons/ArrowDown.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";

export interface EmojiListProps {
  config: string;
  onChange: (value: string) => void;
}

const DND_TYPE = "emoji-row";

interface DraggableEmojiRowProps {
  r: string;
  i: number;
  moveRow: (from: number, to: number) => void;
  configParsed: { [key: string]: string[] };
  inputCache: { [key: number]: string | undefined };
  setInputCache: React.Dispatch<React.SetStateAction<{ [key: number]: string | undefined }>>;
  onChange: (value: string) => void;
  isFirst: boolean;
  isLast: boolean;
  t: (key: string) => string;
}

function DraggableEmojiRow({
  r,
  i,
  moveRow,
  configParsed,
  inputCache,
  setInputCache,
  onChange,
  isFirst,
  isLast,
}: DraggableEmojiRowProps) {
  const ref = React.useRef<HTMLTableRowElement>(null);
  const [, drop] = useDrop({
    accept: DND_TYPE,
    hover(item: any, monitor) {
      if (!ref.current) return;

      const dragIndex = item.index;
      const hoverIndex = i;
      if (dragIndex === hoverIndex) return;

      const hoverBoundingRect = ref.current.getBoundingClientRect();
      const hoverMiddleY = (hoverBoundingRect.bottom - hoverBoundingRect.top) / 2;
      const clientOffset = monitor.getClientOffset();
      if (!clientOffset) return;

      const hoverClientY = clientOffset.y - hoverBoundingRect.top;

      if (dragIndex < hoverIndex && hoverClientY < hoverMiddleY) return;
      if (dragIndex > hoverIndex && hoverClientY > hoverMiddleY) return;

      moveRow(dragIndex, hoverIndex);
      item.index = hoverIndex;
    },
  });
  const [{ isDragging }, drag] = useDrag({
    type: DND_TYPE,
    item: { index: i },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  });
  drag(drop(ref));
  return (
    <TableRow
      ref={ref}
      sx={{ "&:last-child td, &:last-child th": { border: 0 }, opacity: isDragging ? 0.5 : 1, cursor: "move" }}
      hover
    >
      <NoWrapCell>
        <DenseFilledTextField
          fullWidth
          required
          value={r}
          onChange={(e) => {
            const newConfig = {
              ...configParsed,
              [e.target.value]: configParsed[r],
            };
            delete newConfig[r];
            onChange(JSON.stringify(newConfig));
          }}
        />
      </NoWrapCell>
      <NoWrapCell>
        <DenseFilledTextField
          fullWidth
          multiline
          required
          value={inputCache[i] ?? configParsed[r].join()}
          onBlur={() => {
            onChange(
              JSON.stringify({
                ...configParsed,
                [r]: inputCache[i]?.split(",") ?? configParsed[r],
              }),
            );
            setInputCache({
              ...inputCache,
              [i]: undefined,
            });
          }}
          onChange={(e) =>
            setInputCache({
              ...inputCache,
              [i]: e.target.value,
            })
          }
        />
      </NoWrapCell>
      <NoWrapCell>
        <IconButton
          onClick={() => {
            const newConfig = {
              ...configParsed,
            };
            delete newConfig[r];
            onChange(JSON.stringify(newConfig));
          }}
          size={"small"}
        >
          <Dismiss fontSize={"small"} />
        </IconButton>
        <IconButton size="small" onClick={() => moveRow(i, i - 1)} disabled={isFirst}>
          <ArrowDown
            sx={{
              width: "18px",
              height: "18px",
              transform: "rotate(180deg)",
            }}
          />
        </IconButton>
        <IconButton size="small" onClick={() => moveRow(i, i + 1)} disabled={isLast}>
          <ArrowDown
            sx={{
              width: "18px",
              height: "18px",
            }}
          />
        </IconButton>
      </NoWrapCell>
    </TableRow>
  );
}

const EmojiList = memo(({ config, onChange }: EmojiListProps) => {
  const { t } = useTranslation("dashboard");
  const [render, setRender] = useState(false);
  const configParsed = useMemo((): { [key: string]: string[] } => JSON.parse(config), [config]);
  const [inputCache, setInputCache] = useState<{
    [key: number]: string | undefined;
  }>({});
  return (
    <Stack spacing={1}>
      <Box>
        {!render && (
          <SecondaryButton variant={"contained"} onClick={() => setRender(!render)}>
            {t("settings.showSettings")}
          </SecondaryButton>
        )}
        {render && Object.keys(configParsed ?? {}).length > 0 && (
          <DndProvider backend={HTML5Backend}>
            <TableContainer sx={{ mt: 1, maxHeight: 440 }} component={StyledTableContainerPaper}>
              <Table stickyHeader sx={{ width: "100%", tableLayout: "fixed" }} size="small">
                <TableHead>
                  <TableRow>
                    <NoWrapTableCell width={64}>{t("settings.category")}</NoWrapTableCell>
                    <NoWrapTableCell width={200}>{t("settings.emojiOptions")}</NoWrapTableCell>
                    <NoWrapTableCell width={100}>{t("settings.actions")}</NoWrapTableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {Object.keys(configParsed ?? {}).map((r, i, arr) => (
                    <DraggableEmojiRow
                      key={i}
                      r={r}
                      i={i}
                      moveRow={(from, to) => {
                        if (from === to || to < 0 || to >= arr.length) return;
                        const keys = Object.keys(configParsed);
                        const values = Object.values(configParsed);
                        const [movedKey] = keys.splice(from, 1);
                        const [movedValue] = values.splice(from, 1);
                        keys.splice(to, 0, movedKey);
                        values.splice(to, 0, movedValue);
                        const newConfig: { [key: string]: string[] } = {};
                        keys.forEach((k, idx) => {
                          newConfig[k] = values[idx];
                        });
                        onChange(JSON.stringify(newConfig));
                      }}
                      configParsed={configParsed}
                      inputCache={inputCache}
                      setInputCache={setInputCache}
                      onChange={onChange}
                      isFirst={i === 0}
                      isLast={i === arr.length - 1}
                      t={t}
                    />
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </DndProvider>
        )}
      </Box>
      {render && (
        <Box>
          <SecondaryButton
            variant={"contained"}
            startIcon={<Add />}
            onClick={() =>
              onChange(
                JSON.stringify({
                  ...configParsed,
                  [""]: [],
                }),
              )
            }
          >
            {t("settings.addCategorize")}
          </SecondaryButton>
        </Box>
      )}
    </Stack>
  );
});

export default EmojiList;
