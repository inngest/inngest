import { useRouter } from 'next/navigation';
import { useDispatch } from 'react-redux';

import { showDocs } from '@/store/global';

const useDocsNavigation = () => {
  const router = useRouter();
  const dispatch = useDispatch();

  const navigateToDocs = (docsPath) => {
    dispatch(showDocs(docsPath));
    router.push('/docs');
  };

  return navigateToDocs;
};

export default useDocsNavigation;
