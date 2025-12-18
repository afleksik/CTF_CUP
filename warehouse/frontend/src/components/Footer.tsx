export function Footer() {
  return (
    <footer className="mt-auto border-t border-gray-200 bg-white">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="text-center text-gray-600">
          <p className="text-sm">
            © {new Date().getFullYear()} <span className="font-semibold text-gray-900">HazyCorp Team</span>. All rights reserved.
          </p>
        </div>
      </div>
    </footer>
  );
}