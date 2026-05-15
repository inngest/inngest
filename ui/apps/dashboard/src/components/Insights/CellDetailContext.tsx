import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from 'react';

export interface CellDetailData {
  rowIndex: number;
  columnId: string;
  columnType: string;
  value: string | number | Date | null;
}

interface CellDetailContextValue {
  selectedCell: CellDetailData | null;
  openCellDetail: (data: CellDetailData) => void;
  closeCellDetail: () => void;
}

const CellDetailContext = createContext<CellDetailContextValue | null>(null);

interface CellDetailProviderProps {
  children: ReactNode;
  onOpenPanel: () => void;
}

export function CellDetailProvider({
  children,
  onOpenPanel,
}: CellDetailProviderProps) {
  const [selectedCell, setSelectedCell] = useState<CellDetailData | null>(null);

  const openCellDetail = useCallback(
    (data: CellDetailData) => {
      setSelectedCell(data);
      onOpenPanel();
    },
    [onOpenPanel],
  );

  const closeCellDetail = useCallback(() => {
    setSelectedCell(null);
  }, []);

  return (
    <CellDetailContext.Provider
      value={{
        selectedCell,
        openCellDetail,
        closeCellDetail,
      }}
    >
      {children}
    </CellDetailContext.Provider>
  );
}

export function useCellDetailContext(): CellDetailContextValue {
  const context = useContext(CellDetailContext);
  if (!context) {
    throw new Error(
      'useCellDetailContext must be used within a CellDetailProvider',
    );
  }
  return context;
}
