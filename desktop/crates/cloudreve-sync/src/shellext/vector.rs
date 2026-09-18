use std::cell::RefCell;
use windows::Foundation::Collections::{
    IIterable, IIterable_Impl, IIterator, IIterator_Impl, IVector, IVector_Impl, IVectorView,
};

#[windows_core::implement(IVector<T>, IIterable<T>)]
struct StockVector<T>
where
    T: windows_core::RuntimeType + 'static,
    T::Default: Clone + PartialEq,
{
    values: RefCell<Vec<T::Default>>,
}

impl<T> IIterable_Impl<T> for StockVector_Impl<T>
where
    T: windows_core::RuntimeType,
    T::Default: Clone + PartialEq,
{
    fn First(&self) -> windows_core::Result<IIterator<T>> {
        use windows_core::IUnknownImpl;

        Ok(windows_core::ComObject::new(StockVectorIterator {
            owner: self.to_object(),
            current: 0.into(),
        })
        .into_interface())
    }
}

impl<T> IVector_Impl<T> for StockVector_Impl<T>
where
    T: windows_core::RuntimeType,
    T::Default: Clone + PartialEq,
{
    fn GetAt(&self, index: u32) -> windows_core::Result<T> {
        let values = self.values.borrow();
        let item = values
            .get(index as usize)
            .ok_or_else(|| windows_core::Error::from(windows_core::imp::E_BOUNDS))?;
        T::from_default(item)
    }
    fn Size(&self) -> windows_core::Result<u32> {
        Ok(self.values.borrow().len().try_into()?)
    }
    fn GetView(&self) -> windows_core::Result<IVectorView<T>> {
        self.values.borrow().clone().try_into()
    }
    fn IndexOf(&self, value: &T::Default, result: &mut u32) -> windows_core::Result<bool> {
        match self
            .values
            .borrow()
            .iter()
            .position(|element| element == value)
        {
            Some(index) => {
                *result = index as u32;
                Ok(true)
            }
            None => Ok(false),
        }
    }
    fn SetAt(&self, index: u32, value: &T::Default) -> windows_core::Result<()> {
        let mut values = self.values.borrow_mut();
        if let Some(item) = values.get_mut(index as usize) {
            *item = value.clone();
            Ok(())
        } else {
            Err(windows_core::Error::from(windows_core::imp::E_BOUNDS))
        }
    }
    fn InsertAt(&self, index: u32, value: &T::Default) -> windows_core::Result<()> {
        let mut values = self.values.borrow_mut();
        let index = index as usize;
        if index <= values.len() {
            values.insert(index, value.clone());
            Ok(())
        } else {
            Err(windows_core::Error::from(windows_core::imp::E_BOUNDS))
        }
    }
    fn RemoveAt(&self, index: u32) -> windows_core::Result<()> {
        let mut values = self.values.borrow_mut();
        if (index as usize) < values.len() {
            values.remove(index as usize);
            Ok(())
        } else {
            Err(windows_core::Error::from(windows_core::imp::E_BOUNDS))
        }
    }
    fn Append(&self, value: &T::Default) -> windows_core::Result<()> {
        self.values.borrow_mut().push(value.clone());
        Ok(())
    }
    fn RemoveAtEnd(&self) -> windows_core::Result<()> {
        let mut values = self.values.borrow_mut();
        if values.is_empty() {
            Err(windows_core::Error::from(windows_core::imp::E_BOUNDS))
        } else {
            values.pop();
            Ok(())
        }
    }
    fn Clear(&self) -> windows_core::Result<()> {
        self.values.borrow_mut().clear();
        Ok(())
    }
    fn GetMany(&self, current: u32, values: &mut [T::Default]) -> windows_core::Result<u32> {
        let borrowed = self.values.borrow();
        let current = current as usize;
        if current >= borrowed.len() {
            return Ok(0);
        }
        let actual = std::cmp::min(borrowed.len() - current, values.len());
        let (values, _) = values.split_at_mut(actual);
        values.clone_from_slice(&borrowed[current..current + actual]);
        Ok(actual as u32)
    }
    fn ReplaceAll(&self, items: &[T::Default]) -> windows_core::Result<()> {
        let mut values = self.values.borrow_mut();
        values.clear();
        values.extend_from_slice(items);
        Ok(())
    }
}

#[windows_core::implement(IIterator<T>)]
struct StockVectorIterator<T>
where
    T: windows_core::RuntimeType + 'static,
    T::Default: Clone + PartialEq,
{
    owner: windows_core::ComObject<StockVector<T>>,
    current: std::sync::atomic::AtomicUsize,
}

impl<T> IIterator_Impl<T> for StockVectorIterator_Impl<T>
where
    T: windows_core::RuntimeType,
    T::Default: Clone + PartialEq,
{
    fn Current(&self) -> windows_core::Result<T> {
        let current = self.current.load(std::sync::atomic::Ordering::Relaxed);
        let values = self.owner.values.borrow();

        if let Some(item) = values.get(current) {
            T::from_default(item)
        } else {
            Err(windows_core::Error::from(windows_core::imp::E_BOUNDS))
        }
    }

    fn HasCurrent(&self) -> windows_core::Result<bool> {
        let current = self.current.load(std::sync::atomic::Ordering::Relaxed);
        Ok(self.owner.values.borrow().len() > current)
    }

    fn MoveNext(&self) -> windows_core::Result<bool> {
        let current = self.current.load(std::sync::atomic::Ordering::Relaxed);
        let len = self.owner.values.borrow().len();

        if current < len {
            self.current
                .fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        }

        Ok(len > current + 1)
    }

    fn GetMany(&self, values: &mut [T::Default]) -> windows_core::Result<u32> {
        let current = self.current.load(std::sync::atomic::Ordering::Relaxed);
        let borrowed = self.owner.values.borrow();

        let actual = std::cmp::min(borrowed.len() - current, values.len());
        let (values, _) = values.split_at_mut(actual);
        values.clone_from_slice(&borrowed[current..current + actual]);
        self.current
            .fetch_add(actual, std::sync::atomic::Ordering::Relaxed);
        Ok(actual as u32)
    }
}

/// Create an IVector<T> from a Vec<T::Default>
pub fn create_vector<T>(values: Vec<T::Default>) -> windows_core::Result<IVector<T>>
where
    T: windows_core::RuntimeType,
    T::Default: Clone + PartialEq,
{
    Ok(windows_core::ComObject::new(StockVector {
        values: RefCell::new(values),
    })
    .into_interface())
}
