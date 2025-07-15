import { useCallback, useState } from 'react';

export function useReplayModal() {
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [selectedEvent, setSelectedEvent] = useState<{ name: string; data: string } | null>(null);

  const openModal = useCallback((eventName: string, payload: string) => {
    try {
      const parsedData = JSON.stringify(JSON.parse(payload).data);
      setSelectedEvent({ name: eventName, data: parsedData });
      setIsModalVisible(true);
    } catch (error) {
      console.error('Failed to parse event payload:', error);
    }
  }, []);

  const closeModal = () => {
    setIsModalVisible(false);
    setSelectedEvent(null);
  };

  return {
    isModalVisible,
    selectedEvent,
    openModal,
    closeModal,
  };
}
