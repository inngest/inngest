import './index.css'
import Header from './components/Header'
import Sidebar from './components/Sidebar'
import SidebarLink from './components/SidebarLink'
import Main from './components/Main'
import { FeedIcon, BookIcon } from './icons'

export function App() {
  return (
    <div class="w-screen h-screen text-slate-400 text-sm grid grid-cols-app grid-rows-app">
      <Header />
      <Sidebar>
        <SidebarLink icon={<FeedIcon />} active />
        <SidebarLink icon={<BookIcon />} />
      </Sidebar>
      <Main />
    </div>
  )
}
