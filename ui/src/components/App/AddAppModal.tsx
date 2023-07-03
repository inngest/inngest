import Modal from '@/components/Modal';
import Button from '@/components/Button';

export default function AddAppModal({ isOpen, onClose }) {
  return (
    <Modal
      title="Add Inngest App"
      description="Connect your Inngest application to the Dev Server"
      isOpen={isOpen}
      onClose={onClose}
    >
      <div className="bg-[#050911]/50 p-6">
        <p className="text-sm font-semibold text-white">App URL</p>
        <p className="text-slate-500 text-sm pb-4">
          The URL of your application
        </p>

        <input
          className="min-w-[420px] bg-slate-800 rounded-md text-slate-300 py-2 px-4 outline-2 outline-indigo-500 focus:outline"
          placeholder="Your URL"
        />
      </div>
      <div className="flex items-center justify-between p-6 border-t border-slate-800">
        <Button label="Cancel" kind="secondary" btnAction={onClose} />
        {/* To do: Trigger connect action and display error */}
        <Button label="Connect App" btnAction={() => {}} />
      </div>
    </Modal>
  );
}
