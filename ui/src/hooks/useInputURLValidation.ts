import { useEffect, useState } from 'react';

type InputUserUrlValidation = {
  callback?: () => void;
  initialInputValue?: string;
};

function useInputUrlValidation({
  callback = () => {},
  initialInputValue = '',
}: InputUserUrlValidation = {}) {
  const [inputUrl, setInputUrl] = useState(initialInputValue);
  const [isUrlInvalid, setUrlInvalid] = useState(false);

  useEffect(() => {
    let debounce = setTimeout(() => {
      if (inputUrl.length > 0) {
        try {
          new URL(inputUrl);
          setUrlInvalid(false);
          if (callback) {
            callback();
          }
        } catch (err) {
          setUrlInvalid(true);
        }
      } else {
        setUrlInvalid(false);
      }
    }, 500);

    return () => {
      clearTimeout(debounce);
    };
  }, [inputUrl, callback]);

  return [inputUrl, setInputUrl, isUrlInvalid] as const;
}

export default useInputUrlValidation;
